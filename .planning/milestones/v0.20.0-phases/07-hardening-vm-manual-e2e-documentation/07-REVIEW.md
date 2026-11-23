---
phase: 07-hardening-vm-manual-e2e-documentation
reviewed: 2026-11-22T15:00:34Z
depth: standard
files_reviewed: 20
files_reviewed_list:
  - README.md
  - cmd/zcode-acp/main.go
  - cmd/zcode-acp/main_test.go
  - cmd/zcode-acp/testdata/omp-binding-e2e.json
  - docs/provenance.md
  - experiments/README.md
  - internal/acpfront/conformance_test.go
  - internal/acpfront/failurepaths_internal_test.go
  - internal/acpfront/front.go
  - internal/acpfront/lifecycle.go
  - internal/acpfront/lifecycle_internal_test.go
  - internal/acpfront/pickerrefusal_internal_test.go
  - internal/acpfront/sesscompat_test.go
  - internal/acpfront/slashmodel_test.go
  - internal/acpport/agent.go
  - internal/acpport/agent_test.go
  - internal/acpport/types.go
  - internal/interact/keyringprobe.go
  - internal/interact/keyringprobe_classify_internal_test.go
  - scripts/vm-e2e.sh
findings:
  critical: 0
  warning: 3
  info: 8
  total: 11
  necessity: 0
status: issues_found
---

# Phase 7: Code Review Report

**Reviewed:** 2026-11-22T15:00:34Z
**Depth:** standard
**Files Reviewed:** 20
**Status:** issues_found

## Summary

Reviewed all 20 Phase 7 files at standard depth: the omp dispatch/binding surface in `cmd/zcode-acp`, the SESS-02/SESS-03 front contracts (`front.go`, `lifecycle.go`), the acpport adapter (`agent.go`, `types.go`), the keyring probe (`keyringprobe.go`), the VM e2e script, both doc surfaces, and the phase's test files. Cross-file traces were run where the narrative findings depend on them (recordSettings call sites, acquireTurn's closing gate, resume reply shape, doc link targets, CI workflow claims).

The implementation quality is high overall: the omp argv dispatch is test-locked and value-token safe, the SESS-01 teardown ordering and lock discipline hold up under trace, the wire byte-locks are consistent between code and tests, and every doc claim I checked against the repo (workflow files, ported-file list, probe timings, version stamps, close budget) verifies. No critical defects were found.

Three warnings remain. The two front-end ones are the substance of this review: the `session/resume` path never records the confirmed settings snapshot the backend just returned, which breaks the picker and the `/model` slash switch on resumed sessions; and the `/model` slash intercept is deliberately placed before the turn-slot acquisition with no slot-availability check of its own, so a switch issued while a turn runs bypasses the CORE-05 rejection and — on a backend refusal — runs the WR-01 teardown that cancels the running turn and destroys the session. The third warning is credential hygiene: the `--api-key` flag places the key in the world-readable process argv.

Necessity review (`$HOME/.zcode/gsd-core/references/necessity-review.md` was read per the standard-depth contract): 0 necessity findings. The deliberate fail-loud "unreachable" arms in `setConfigMutation` sit on a client-input trust boundary (negative space, not impossible-state guards), the `closing` map lazy-init is load-bearing (`New` does not initialize it), and the probe's parent-cancel/deadline ordering is behavior under test, not cargo cult.

## Warnings

### WR-01: Session/resume drops the confirmed settings snapshot — picker and /model are dead on resumed sessions

**File:** `internal/acpfront/lifecycle.go:64-119`
**Issue:** `ResumeSession` digs the session id out of the backend `session/resume` reply via `createSessionID` — whose own contract (front.go:1195-1199) states session/create and session/resume carry the SAME full snapshot shape — records the cwd, and then discards the reply's `settings` snapshot. `recordSettings` is called from exactly four sites (front.go:289 create reply, 612 and 726 mutation replies, 941 setModel follow-up), none on the resume path. Consequences for every resumed session: `SetSessionConfigOption` answers -32602 "no confirmed settings to validate against (unknown session)" (misleading — the session IS registered), and the `/model` slash prompt answers the pinned refusal `No model selected.` even when the backend just confirmed a bound model in the resume reply. This contradicts the D-10 posture the store exists for (configOptions built ONLY from confirmed reply snapshots — the resume reply IS one) and breaks the phase's own SESS-03 feature on the resume leg.
**Fix:** Mirror `NewSession`'s capture in `ResumeSession`, right beside the cwd record:

```go
f.recordCwd(sid, cwd)
f.recordSettings(sid, res) // the resume reply IS the confirmed snapshot (D-10)
```

and drop both in the subscribe-failure path beside the existing `f.dropCwd(sid)` (the WR-07 bounding invariant this phase's failurepaths test already locks for cwd).

### WR-02: /model slash intercept bypasses the single-active-turn guard — a refused mid-turn switch cancels the running turn and tears the session down

**File:** `internal/acpfront/front.go:365-368,702-722`
**Issue:** `promptModelSlash` runs BEFORE `acquireTurn` (deliberately, so refusals never hold the slot), but nothing re-checks slot availability for the successful path. A `/model` prompt issued while a turn of the same session is active therefore (a) skips the `turnActiveMessage` rejection any literal prompt would get and issues a backend `session/setModel` mutation mid-turn, and (b) if the backend refuses it, runs `teardownAfterPickerRefusal` → `CloseSession` → `drainActiveTurn`, which cancels the user's RUNNING turn (it resolves `cancelled`) and tears the whole session down as a side effect of a refused switch. The single-active-turn invariant (D-05/CORE-05) exists precisely because clients do send concurrent prompts; the intercept must classify the busy case instead of racing the slot.
**Fix:** Gate the mutating arm on slot availability before the `conn.Call`, using the same rejection surface as Prompt:

```go
f.turnMu.Lock()
busy := f.turnActive && f.turnSession == params.SessionID
f.turnMu.Unlock()
if busy {
    return "", true, acpport.NewInvalidRequest(turnActiveMessage)
}
```

(cross-session turns stay unaffected — the check is session-scoped, matching `Cancel`'s WR-07 discipline). Alternatively route the applied switch through `acquireTurn` so it serializes with turns by construction.

### WR-03: --api-key rides the process argv — world-readable via ps//proc for the wrapper's lifetime

**File:** `cmd/zcode-acp/main.go:98-99`
**Issue:** The `--api-key` flag value is part of the wrapper's command line, which Linux makes world-readable (`ps aux`, `/proc/<pid>/cmdline`) for the process's whole lifetime — on a multi-user host every local user can read the key. This contradicts the project's own credential-exposure discipline: the README launch recipe (README.md:125-128) praises the client tokenizer for rejecting `VAR=val cmd` so "an API key can never ride the override string", and T-07-20 mandates env-var-names-only — yet the README configuration table (README.md:43) advertises `--api-key` with no exposure caveat, and argv is strictly worse than the mechanisms the docs steer users to (env/environ is owner-only readable).
**Fix:** Keep the flag for compatibility but (1) document the process-list exposure at the README table entry and steer to `ZCODE_ACP_API_KEY` / `agentDefaultEnv` as the recommended mechanism, and (2) optionally log a warning when the flag form is used. A fuller fix drops secret-bearing argv entirely and accepts the key from a `--api-key-file` descriptor.

## Info

### IN-01: classify_sentinel's "ambiguous" arm is dead — a multi-sentinel reply is misreported as "no sentinel matched"

**File:** `scripts/vm-e2e.sh:503-525,600-602`
**Issue:** The function documents `"ambiguous" for >1 match`, but the body only echoes when `matched -eq 1`; a 2-match reply echoes nothing, so the caller's `ambiguous)` case arm is unreachable dead code and the failure is recorded with the wrong reason ("no table sentinel matched the reply") — degrading the diagnosis the check exists to produce.
**Fix:** Add `if [ "$matched" -gt 1 ]; then printf 'ambiguous'; return 0; fi` before the single-match echo (the `ambiguous)` arm then works as written).

### IN-02: --stage nokeyring-arm is accepted on the public CLI and answers a vacuous PASS

**File:** `scripts/vm-e2e.sh:142-148,856,860-865`
**Issue:** The internal arm's stage name is accepted by the public stage validator; a user invoking `--stage nokeyring-arm` runs the arm with no `--internal-arm` guard, records zero checks, and prints `vm-e2e: PASS (0 checks, 0 skipped)` — a false green from a QA deliverable whose contract is honest verdicts.
**Fix:** Reject `nokeyring-arm` when `INTERNAL_ARM -ne 1` (usage error, exit 2).

### IN-03: Shutdown select races client-disconnect against child-exit — a coincident child crash can exit 0

**File:** `cmd/zcode-acp/main.go:683-690`
**Issue:** `select` over `agentConn.Done()` and `child.Done()` picks randomly when both are ready; if the child died but the client-disconnect arm wins, `childLost` stays false and the process exits 0 ("shutdown complete") instead of the honest `errChildLost` exit 1. The exit-code classes are a documented contract of this binary (README.md:50-51).
**Fix:** After the disconnect arm wins, re-check `child.Done` non-blockingly (`select { case <-child.Done: childLost = true; default: }`) before the success path.

### IN-04: Unknown-flag tolerance is coupled to the stdlib flag error string

**File:** `cmd/zcode-acp/main.go:303-305`
**Issue:** `isUnknownFlagError` matches `errors.Is`-style on the message `"flag provided but not defined"`. The omp card's tolerance posture (its whole acceptance contract for future client flags) silently degrades to hard exit-2 failures if a Go release ever rewords the message.
**Fix:** Pre-scan the chunk's tokens against the known flag-name set from `ompFlagSet` and treat unknown dash-tokens as tolerated before handing the chunk to `fs.Parse`; keep the message match only as a fallback.

### IN-05: ResumeSession trusts the reply's session id without an equality check

**File:** `internal/acpfront/lifecycle.go:93-102`
**Issue:** The front registers cwd (and, after WR-01's fix, settings) under `sid` dug from the reply, while the adapter discards the front-returned id (agent.go:466-483) — the client only knows the id it requested. A backend reply carrying a divergent id would key all front state under an id the client never uses (its close/prompt then fail unknown) with no coded signal at resume time.
**Fix:** Compare `sid != sessionID` after `createSessionID` and fail with a coded internal error naming both ids.

### IN-06: DeleteSession's silent-success arm has a concurrent-teardown race window

**File:** `internal/acpfront/front.go:640-649`
**Issue:** `DeleteSession` does `registry.Get`, then `CloseSession` re-does it. A teardown landing between the two gets the second Get to fail, and the delete answers the close-in-progress/unknown coded error instead of the spec's silent success `{}` for an already-closed id (the exact case sesscompat_test.go:243-277 locks on the sequential path).
**Fix:** Inside `DeleteSession`, treat `CloseSession`'s unknown-id coded invalid-request as success (the registry miss after a passing Get proves a concurrent teardown completed), or fold the mark-and-teardown so the delete owns the idempotence check.

### IN-07: agent_expect bounds each single read, not the leg — a steadily streaming wrapper can exceed the TIMEOUT_* budgets

**File:** `scripts/vm-e2e.sh:265-285`
**Issue:** `read -t "$bound"` resets per line, so a wrapper that keeps emitting frames (a hung turn that still streams deltas) never trips the documented leg bound — the header's "a stage never hangs" contract holds only for silent hangs. The no-keyring arm is `timeout(1)`-wrapped; the run and auth arcs are not.
**Fix:** Record a leg-start timestamp and break the read loop (FAIL) when it exceeds the bound, independent of per-line progress.

### IN-08: --help truncates the output-contract and exit-codes sections

**File:** `scripts/vm-e2e.sh:131-134`
**Issue:** `--help` prints header lines 2-60 only; the documented output contract and exit-code tables (lines 66-81) — the part a consumer of the script's output needs — are cut off.
**Fix:** `sed -n '2,81p' "$SELF"` (or compute the end from the header sentinel).

---

_Reviewed: 2026-11-22T15:00:34Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
