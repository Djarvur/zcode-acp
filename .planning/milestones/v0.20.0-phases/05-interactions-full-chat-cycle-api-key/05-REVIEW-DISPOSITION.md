---
phase: "5"
slug: "interactions-full-chat-cycle-api-key"
review: 05-REVIEW.md
status: issues_found
critical: 1
warning: 4
info: 3
total: 8
updated: "2026-11-21"
---

# Phase 5 — Review Disposition

> Per-finding disposition ledger for `05-REVIEW.md` (advisory, never blocks).

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | `drainActiveTurn` returns false after a successful drain — `CloseSession` skips the permission-bridge cancel fan-in exactly on the close-with-active-turn path; cancelled turn's pending permission dangles to the 300s responder deadline then frames a deny at a closed backend session. In-scope for the phase goal (permission decisions answered from the client); must be fixed and test-covered (close-with-pending-permission) before REAL RUN UAT item on permissions can be trusted |
| WR-01 | warning | open | Parked dedupe-join arm frames a bare id-echo reply at conn close, contradicting Serve's never-send invariant (`responder.go` + `conn.go` bypass lane window) |
| WR-02 | warning | open | `ResumeSession` drops the client cwd (SDK `ResumeSessionRequest.Cwd` exists, v0.20.0) — resumed sessions get no `cwds` entry, relative diff/location paths (Pitfall-5 on the resume path) |
| WR-03 | warning | open | `session/resume` not gated on the closing set — close-vs-resume race leaks a machine's conn observers (WR-07-class from 04-REVIEW) |
| WR-04 | warning | open | Prompt's cancel fan-in goroutine can wake on both ctx arms and cancel a later, unrelated turn on the same machine — needs arm ordering synchronized with `release()` |
| IN-01 | info | open | realdesktop `resolveModelSelection` silently ignores malformed `ZCODE_ACP_MODEL` (production exits 2 on the same shape) — real spend on an unintended model |
| IN-02 | info | open | raw-capture path not enforced outside the repo — the tee holds the live api-key; "born OUTSIDE" rule lives only in comments |
| IN-03 | info | open | main's shutdown select can mask a child crash with exit 0 when it races client disconnect |

**Open:** 8 of 8 (CR-01 gates the REAL RUN UAT retry; WR-01..04 batch with the CR-01 close-path fix; IN-01..03 hardening backlog) · **Fixed:** 0 · **Accepted:** 0

**Rationale:** Fresh deep review of the 42-file phase-5 scope found one critical correctness inversion on the close/permission path the phase goal explicitly owns. The phase is already stopped on human UAT (REAL RUN attempt 1: auth read-back gap); CR-01 joins the pre-UAT fix list. Ledger stays open so the gap-closure/UAT fix plan MUST cite it.

---

## Delta re-review (2026-11-21, post gap-closure cycles, base af5a1ed..HEAD, 23 files)

Status `issues_found`: 0 critical, 3 warnings, 4 infos. Prior findings disposition after first-hand verification (`-race` suites green, golden corpus 3x, scan-secrets 0):

| Prior finding | Disposition |
|---------|----------|
| CR-01 | **fixed** — sawTurn drain semantics + close-with-pending-permission lock test (05-10); wire-locked incl. the deadline arm |
| WR-01 | **fixed** — structural nil-nil join silence at the reply funnel (05-10) |
| WR-02 | **fixed** — ResumeSession cwd seam end-to-end incl. zero-wire-frame parity arm (05-10) |
| WR-03 | **superseded → WR-05** — closing gate landed but narrowed: TOCTOU remains (close completing between resume's entry gate and re-check, or Register landing in close's Get→mark gap, still leaks observers/zombie); the fix's "no leak" doc overclaims |
| WR-04 | **fixed** — pump-identity re-check sound; release nils the slot under the same lock |
| IN-01..IN-03 | open (hardening backlog, unchanged) |

New findings (delta code):

| Finding | Severity | Disposition | Note |
|---------|----------|-------------|------|
| WR-05 | warning | open | resume's closing gate TOCTOU (entry-gate→re-check window; close's Get→mark gap) — observers/zombie leak arm remains |
| WR-06 | warning | open | displaced-prompt fan-in arm cancels the machine but never fans the turn's pending permission bridge in — dangles to the 300s deadline then answers deny for a cancelled turn (TOOL-02 on that arm); `TestCancelFanInOnlyWhenSlotHeld`/`assertDanglingAskNote` currently enshrine the dangling artifact |
| WR-07 | warning | open | Cancel's observe-before-cancel sweep unguarded — deschedule across a turn boundary can cancel a later session's fresh ask; fix shape: session-scoped pendings |
| IN-04 | info | open | single-shot probe before Await — late-applying push on a patchless desktop never latches (honest failure, latency gap) |
| IN-05 | info | open | fake arms the latch after the reply write — dispatch race can slip send-2 ahead of the arm (loud golden flake) |
| IN-06 | info | open | realdesktop ask tap calls t.Logf from conn goroutines — mid-traffic fatal can panic post-test |
| IN-07 | info | open | readbackdiag capture opener repeats IN-02's born-OUTSIDE class (second call site for the IN-02 fix) |

**Open:** WR-05..07 + IN-01..07 (10) · **Fixed:** CR-01, WR-01, WR-02, WR-04 · **Superseded:** WR-03 → WR-05

**Rationale (delta):** the gap-fix code closed all four pre-UAT findings with structural locks and survives first-hand -race/golden/secret verification; the remaining warnings are narrower arms (resume TOCTOU, displaced-prompt fan-in, cancel sweep deschedule) — real debt, none observed in the field-proven attempt-3 run; they feed the next hardening cycle, not a verdict block on their own.
