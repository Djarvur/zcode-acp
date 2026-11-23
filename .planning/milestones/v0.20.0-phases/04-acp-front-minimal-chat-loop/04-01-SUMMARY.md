---
phase: 04-acp-front-minimal-chat-loop
plan: 01
subsystem: infra
tags: [acp, go-modules, jsonrpc, observer-pattern, sdk, supply-chain]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: "zcode.Conn codec (demux, deliverNotification), turn.Machine (ObserveEnvelope, NewMachine hook wiring), fake app-server + golden corpus, supervisor exec chain"
provides:
  - "github.com/coder/acp-go-sdk v0.20.0 pinned as the module's sole external dependency (go.mod/go.sum), held tidy-stable by the acpport seam pin and enforced by the hardened tidy-diff gate"
  - "zcode.Conn.AddObserver(fn) (remove func()) — ordered notification fan-out replacing the single OnNotification slot (registration order, synchronous on the read goroutine, wire order preserved, idempotent remove)"
  - "zcode.MethodSessionSubscribe / zcode.MethodSessionSetModel codec constants (dialect §2 method map, D-06 naming)"
  - "internal/acpport seam package skeleton (doc.go package contract + pin_test.go tidy-stability pin) — the sole sanctioned SDK import home for plans 04-02+"
  - "TestObserverFanOut contract suite: registration-order delivery, synchronous read-goroutine proof, remove, two-machine interleaved exactly-once via terminal-event signal"
affects: [04-02, 04-03, 04-04, 04-05, 04-06, phase-5-chat-cycle, phase-6-auth]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 6845 # chars/4 over the 12 production/config files changed (27379 diff chars)
  tasks: 2
  commits: 3 # MEASURED: git rev-list --count 0db91db..HEAD

commits: 3
plan_head_before: 0db91dbadc0c4fec8fdc4ef967a94d300de62b37
plan_head_after: 9d606720ba4f2a9314f2f67a0307b88fc97b0a54

# Tech tracking
tech-stack:
  added:
    - "github.com/coder/acp-go-sdk v0.20.0 (exact pin — D-01 audit PASS: Apache-2.0, coder org, zero transitive deps, daily-active maintenance)"
  patterns:
    - "Observer fan-out at the conn boundary: ordered observer list, deliveries synchronous in registration order on the read goroutine, pointer-identity remove funcs (function values are not comparable)"
    - "Dependency pin discipline: tidy-diff gate asserts the expected require set (count + fixed-string pin), not just zero drift"
    - "Tidy-stable pin via a blank import in a seam-package test file (go mod tidy retains test-only imports); boundary tests exempt _test.go"

key-files:
  created:
    - internal/acpport/doc.go
    - internal/acpport/pin_test.go
    - go.sum
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-01-red-observer-fanout.json
  modified:
    - go.mod
    - mise.toml
    - internal/zcode/envelope.go
    - internal/zcode/conn.go
    - internal/zcode/conn_test.go
    - internal/turn/machine.go
    - internal/zcode/fake/golden_test.go
    - internal/zcode/fake/roundtrip_test.go
    - internal/supervisor/e2e_test.go

key-decisions:
  - "acpport seam package pre-placed one plan early (doc.go + pin_test.go blank SDK import): go mod tidy drops unimported requires, so the gate's exactly-one-require expectation needs an import somewhere; the pin lives INSIDE the sanctioned seam package so D-01's seam lock and 04-02's TestSDKImportBoundary (which exempts _test.go) both hold [Rule 3]"
  - "tidy-diff gate hardened to assert the D-01 dependency expectation (exactly one ^require line, fixed-string pin to acp-go-sdk v0.20.0) with a Phase-5 comment naming the godbus extension obligation (T-4-SC pin discipline)"
  - "OnRouterEvent stays a single slot by design in Phase 4 — one active turn per front; Phase 5 CORE-06 revisits (per plan)"
  - "Two-machines subtest pins the RESOLVING SIGNAL (turn.SignalTerminalEvent), not just the resolution: the per-machine poll watcher is never stolen, so a stolen hook still resolves via the quiet window — the signal assertion is the slot-theft fingerprint"

patterns-established:
  - "zcode.Conn.AddObserver(func(zcode.Envelope)) (remove func()) — the notification observation surface every front-tier component (delta tap, machines) registers on"
  - "Hand-scripted peer pattern for multi-machine conn tests: autoAnswerPeer echoes type-tagged ids verbatim and answers send acks + session/messages pages"
  - "Param-map dig for camelCase wire keys under the tagliatelle contract (machine.go classifyIdlePatch precedent, reused in autoAnswerPeer)"

requirements-completed: [CORE-01, CORE-02, CORE-03, CORE-04, CORE-05] # copied verbatim from plan frontmatter; delivery is shared across plans 04-02..04-06 — requirements.ready-ids returned [] here, so NOTHING is marked complete by this plan

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "github.com/coder/acp-go-sdk pinned at v0.20.0 as the module's sole external dependency; mise tidy-diff gate fails on any require drift, wrong require count, or missing pin"
    verification:
      - kind: other
        ref: "mise run tidy-diff (go mod tidy + git diff --exit-code + grep -c '^require' == 1 + fixed-string pin)"
        status: pass
    human_judgment: false
  - id: D2
    description: "MethodSessionSubscribe = \"session/subscribe\" and MethodSessionSetModel = \"session/setModel\" codec constants exist per D-06 naming (dialect §2 method map)"
    verification:
      - kind: other
        ref: "grep -c 'MethodSession\\(Subscribe\\|SetModel\\)' internal/zcode/envelope.go == 2; go build ./... && go vet ./..."
        status: pass
    human_judgment: false
  - id: D3
    description: "Ordered notification observer fan-out in zcode.Conn: registration-order synchronous delivery on the read goroutine, idempotent remove, two machines on one conn each resolving interleaved turns exactly once via the terminal-event signal; OnNotification slot fully removed"
    verification:
      - kind: unit
        ref: "internal/zcode/conn_test.go#TestObserverFanOut (ordering / remove / two-machines, -race)"
        status: pass
      - kind: other
        ref: "grep -c 'OnNotification' internal/zcode/conn.go == 0 (was 5 at HEAD 83926cd)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Zero regressions: full internal suite (zcode conn/writer/router, fake golden corpus, roundtrip, supervisor e2e, turn machine, acpport) green under -race on the branch head"
    verification:
      - kind: integration
        ref: "mise run ci (build + vet + lint + go test -race -count=1 ./... + node --test experiments/)"
        status: pass
    human_judgment: false

# Metrics
duration: 20min
completed: 2026-11-20
status: complete
---

# Phase 4 Plan 01: Minimal Chat Loop Plumbing Summary

**acp-go-sdk v0.20.0 pinned as the sole external dependency behind a hardened tidy-diff gate, plus ordered notification observer fan-out in zcode.Conn replacing the single-slot OnNotification hook (Pitfall 3 fix) and the subscribe/setModel method constants**

## Performance

- **Duration:** 20 min
- **Started:** 2026-11-19T23:25:44Z
- **Completed:** 2026-11-19T23:45:34Z
- **Tasks:** 2 (Task 2 via full RED→GREEN TDD)
- **Files modified:** 12 production/config files + 1 evidence record

## Accomplishments
- Dependency set frozen at exactly one approved require line: `github.com/coder/acp-go-sdk v0.20.0` (D-01 audit PASS), with the mise tidy-diff gate now asserting the expected dependency set — any unapproved require fails every later gate (T-4-SC mitigation landed)
- `zcode.Conn.AddObserver` ordered fan-out shipped test-first: notifications deliver to ALL observers synchronously in registration order on the read goroutine (wire order preserved), remove funcs deregister idempotently, and the `OnNotification` single slot is gone from conn.go (all five HEAD occurrences)
- `turn.NewMachine` registers via the fan-out; two machines on one conn each resolve their own interleaved turns exactly once — a second machine can no longer steal the first's notification hook
- `MethodSessionSubscribe` / `MethodSessionSetModel` constants landed for plans 02+; `mise run ci` fully green (build, vet, lint 0 issues, full -race suite, experiments)

## Task Commits

Each task was committed atomically (Task 2 is a TDD feature: RED commit precedes GREEN):

1. **Task 1: Land the D-01 SDK dependency + tidy-diff gate expectation + subscribe/setModel method constants** - `7bad5b5` (feat)
2. **Task 2 (RED): observer fan-out contract tests failing on delivery assertions** - `4829602` (test)
3. **Task 2 (GREEN): ordered notification observer fan-out + machine registration migration** - `9d60672` (feat)

**Plan metadata:** committed in the close-out docs commit (this file + STATE/ROADMAP/REQUIREMENTS).

## TDD Record

- **RED:** TestObserverFanOut (conn_test.go) written against a storage-only `AddObserver` (compiles, delivers nothing). Evidence: `evidence/04-01-red-observer-fanout.json` — classifier verdict `RED_EVIDENCE_OK` (go-test adapter, target matched, exit 1). All three subtests failed on delivery assertions: ordering (second envelope consumed while the first observer was "parked" — no observer ever called), remove (kept observer never fired within 10s), two-machines (sess_a resolved via signal `quiet-window` instead of `terminal-event`). **Semantic assessment:** genuine RED on the planned assertion, not a build/fixture failure; recorded nuance — the two-machines turn still resolves exactly-once in RED via the machine-owned poll watcher's quiet-window fallback, which is why the subtest asserts the resolving SIGNAL (terminal-event) to make hook theft observable.
- **GREEN:** `deliverNotification` iterates the ordered observer list synchronously under the obsMu lock; the OnNotification field, its doc comment, the deliverNotification hook reference, and the nil-check+dispatch pair were all removed; machine.go and the four test files migrated to `AddObserver`. Target test and the full `./internal/...` suite green under `-race`; golangci-lint 0 issues.
- **REFACTOR:** not needed (the GREEN commit already carries the funcorder/gocognit/funlen/lll/tagliatelle-compliant shape).

## Files Created/Modified
- `go.mod` / `go.sum` - exactly one require line pinned at v0.20.0 + two module hashes (zero transitive deps)
- `mise.toml` - tidy-diff gate asserts the dependency expectation (count == 1 + fixed-string pin; Phase-5 godbus comment)
- `internal/zcode/envelope.go` - MethodSessionSubscribe / MethodSessionSetModel appended to the dialect §2 method table
- `internal/zcode/conn.go` - AddObserver + observerEntry (pointer-identity remove), ordered observer delivery in deliverNotification, OnNotification slot removed
- `internal/zcode/conn_test.go` - TestObserverFanOut (ordering / remove / two-machines) + autoAnswerPeer hand-scripted peer + spy helpers; both legacy OnNotification call sites migrated
- `internal/turn/machine.go` - NewMachine registers via conn.AddObserver (OnRouterEvent stays single-slot by design); comments updated
- `internal/zcode/fake/golden_test.go` / `internal/zcode/fake/roundtrip_test.go` - notification hook assignments migrated to AddObserver
- `internal/supervisor/e2e_test.go` - hook-contract comment updated to the registration model
- `internal/acpport/doc.go` / `internal/acpport/pin_test.go` - seam package skeleton: package contract + tidy-stability pin (Rule 3 deviation, see below)

## Decisions Made
- Two-machines subtest asserts the resolving signal (`turn.SignalTerminalEvent`), not just exactly-once resolution — the machine-owned poll watcher would otherwise mask hook theft behind a quiet-window resolution (discovered during RED)
- The remove func matches by `*observerEntry` pointer identity (function values are not comparable in Go), making double-removes and index shifts safe
- Deliveries iterate the observer list while holding obsMu: the hook contract (register before any frame flows) means the lock never contends with in-flight registration on the real path

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] acpport seam package skeleton added so the SDK require survives `go mod tidy`**
- **Found during:** Task 1 (dependency landing)
- **Issue:** the plan says "Do NOT import the SDK anywhere yet" while also hard-gating `mise run tidy-diff` green with exactly one require line pinned at v0.20.0. Empirically (verified before any code change): `go get` adds the require, but `go mod tidy` DROPS unimported modules — the committed require line would be deleted by the very gate that must preserve it. The gate as specified cannot pass without an import somewhere in the module.
- **Fix:** pre-placed the `internal/acpport` seam package skeleton one plan early: `doc.go` (package contract in the machine.go decision-ID comment style) + `pin_test.go` (blank `_ "github.com/coder/acp-go-sdk"` import). `go mod tidy` retains requirements imported by any test in the module, so the pinned require is tidy-stable without the `// indirect` marker. The import lives INSIDE the sanctioned seam package, so D-01's seam lock holds and plan 04-02's TestSDKImportBoundary is unaffected (it exempts `_test.go` files and permits imports inside internal/acpport).
- **Files modified:** internal/acpport/doc.go, internal/acpport/pin_test.go
- **Verification:** `go get` → `go mod tidy` → require line and go.sum hashes stable; `mise run tidy-diff` green with the new expectation; `go build ./... && go vet ./...` clean; recorded in the windows ledger (entry 8, kind stub, resolving plan 04-02)
- **Committed in:** 7bad5b5 (Task 1 commit)

**2. [Rule 1 - Bug] Two-machines subtest was not theft-sensitive in RED (resolved pre-evidence)**
- **Found during:** Task 2 RED iteration (before the evidence record was captured)
- **Issue:** the first RED run showed TestObserverFanOut/two-machines PASSING under the stolen-slot condition: the per-machine poll watcher (never stolen) resolved the turn via its quiet-window fallback, hiding the stolen notification hook.
- **Fix:** the subtest now additionally asserts the resolution's signal is `turn.SignalTerminalEvent`; a quiet-window resolution fails the subtest with the theft fingerprint. Re-ran RED: all three subtests fail on their planned assertions; only then was the evidence record captured and the classifier gate run.
- **Files modified:** internal/zcode/conn_test.go
- **Verification:** RED run (exit 1, three failing subtests) → RED_EVIDENCE_OK → GREEN run green
- **Committed in:** 4829602 (RED commit carries the final test shape)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** both fixes required for the plan's own gates to hold (the tidy-diff expectation and a discriminating RED). No scope creep; the plan's prohibitions hold — the SDK is imported ONLY inside internal/acpport, and only from a test file.

## Issues Encountered
- `mise run tidy-diff` compares against HEAD, so it necessarily failed until the dependency files were committed — verification order for Task 1 ran the other checks first, committed, then confirmed the gate green on the committed tree.
- golangci-lint (strict v2.15.0) flagged six classes on the new test code (funcorder, gocognit, funlen, lll, tagliatelle); all fixed in the GREEN commit — AddObserver moved after Close, subtests extracted to top-level funcs, param map-dig for the camelCase `sessionId` wire key (machine.go precedent).

## Known Stubs

| Stub | File | Reason | Resolved by |
|------|------|--------|-------------|
| acpport seam package is a skeleton (package contract + pin only; no adapter surface) | internal/acpport/doc.go:12 | intentional pre-placement so the D-01 pin is tidy-stable from day one | plan 04-02 (types.go/agent.go/sink.go + TestSDKImportBoundary) |

Recorded in the cross-phase defect register: `.planning/WINDOWS.md` entry 8 (open).

## User Setup Required

None - no external service configuration required (module proxy fetch only; no quota cost).

## Next Phase Readiness
- Plans 04-02..04-06 can compile against the pinned SDK and the new codec constants; the fan-out supports the front-tier delta tap (register before the machine, RESEARCH Pattern 3 ordering) without slot theft across ACP sessions
- The tidy-diff gate now hard-locks the dependency set: plans 02+ must land any new dependency ONLY via the audit path plus a gate-expectation extension (Phase 5 godbus comment already names the obligation)
- No blockers; the tracer plan (04-06) composes directly on this plumbing

---
*Phase: 04-acp-front-minimal-chat-loop*
*Completed: 2026-11-20*

## Self-Check: PASSED

All created files exist on disk; all three task commits (7bad5b5, 4829602, 9d60672) are ancestors of HEAD (9d60672). Commit count measured from the persisted plan ledger: 3 (0db91db..HEAD).
