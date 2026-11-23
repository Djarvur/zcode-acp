# Phase 6 — Review Disposition Ledger

> One row per finding from [06-REVIEW.md](./06-REVIEW.md) (0 Critical / 6 Warning / 4 Info).
> Default disposition: `open`. Advisory artifact — never blocks; the traceable answer to "what happened to WR-NN".
> Written by the execute-phase code_review_gate, 2026-11-22.

| Finding | Severity | Disposition | Rationale / Route |
|---------|----------|-------------|-------------------|
| WR-01 refused picker setModel leaves front session alive vs dead backend | warning | **open** → Phase 7 | Session-lifecycle error-path consistency — the Hardening phase's domain (SESS tier). Same class as phase-4's routed WR/IN set. Fix needs a designed teardown contract, not a drive-by. |
| WR-02 concurrent runtime-headers asks in the D-05 re-mint window answer misleading login_required | warning | **resolved** `4dbe1aa` | Cache-miss arm routes through the single-flight Ensure — riders join the in-flight mint instead of answering from a known-empty cache; D-05 contract untouched (RED `d5e8c22`, deterministic under -race ×3). |
| WR-03 testPatConcurrentArm flake across the invalidate→ensure window | warning | **resolved** `d5e8c22` | scriptMinter single-flight + ensureHold pin: 7 riders verified to join the parked mint; exactly 1 Invalidate + 1 Ensure; deterministic (~0.01s). |
| WR-04 three post-create failure paths leak cwd/settings map entries (WR-07 invariant) | warning | **open** → Phase 7 | Bounding-invariant cleanup on failure paths; mechanical but touches lifecycle semantics shared with WR-01/WR-05 — settle as one Phase-7 teardown pass. |
| WR-05 resume/close TOCTOU on isClosing re-check + registry.Register | warning | **open** → Phase 7 | Narrow interleaving in the same lifecycle code as WR-01/WR-04; one designed fix pass in Phase 7 rather than three risky spot-patches. |
| WR-06 cpAskRecorder missing mutex (data race on the manual evidence run) | warning | **resolved** `d5e8c22` | cpAskRecorder carries sync.Mutex over recordAsk/recordHeaderAnswer/lastAskAt + headerAskShapes() snapshot accessor, mirroring the sibling askRecorder guard; real-run harness now race-clean. |
| IN-01 builtin-table double-read (stale revision/rules pairing) | info | **open** → Phase 7 | Advisory; correctness nuance, no field symptom. |
| IN-02 adapter comment claims validation agent.go does not perform | info | **open** → Phase 7 | Comment-accuracy fix, batched with Phase-7 doc pass. |
| IN-03 deadline-boundary classification race in keyringprobe | info | **open** → Phase 7 | Advisory; boundary classification only, fail-fast semantics unaffected. |
| IN-04 dead settings==nil re-init guard (necessity) | info | **open** → Phase 7 | Dead-code removal, batched with Phase-7 cleanup. |

**Security pass:** no Critical; argv/user-input paths clean; credentials-never-logged re-confirmed structural.

**Summary:** 3 fix-in-phase (WR-02, WR-03, WR-06 — concurrency/pre-real-run), 7 open → Phase 7 (Hardening owns error-path consistency + advisory cleanup), 0 resolved-in-phase at write time.
