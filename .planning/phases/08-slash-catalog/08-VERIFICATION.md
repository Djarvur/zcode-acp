---
phase: 08-slash-catalog
verified: 2026-11-23T11:17:28Z
status: passed
score: 5/5 must-haves verified
covered_files:
  - .planning/phases/08-slash-catalog/08-01-PLAN.md
  - .planning/phases/08-slash-catalog/08-01-SUMMARY.md
  - README.md
  - cmd/zcode-acp/main.go
  - docs/protocol/acp-client-dialect.md
  - experiments/captures/v021-catalog-vm-menu.png
  - internal/acpfront/catalog.go
  - internal/acpfront/catalog_emit_test.go
  - internal/acpfront/catalog_internal_test.go
  - internal/acpfront/front.go
  - internal/acpfront/harness_test.go
  - internal/acpfront/reconcile_stream_test.go
  - internal/acpfront/stream_order_test.go
  - internal/acpfront/testdata/catalog-cmds/broken.md
  - internal/acpfront/testdata/catalog-cmds/gsd-manager.md
  - internal/acpfront/testdata/catalog-cmds/plain-cmd.md
  - internal/acpfront/testdata/catalog/alpha-skills/SKILL.md
  - internal/acpfront/testdata/catalog/broken/SKILL.md
  - internal/acpfront/testdata/catalog/not-a-skill/README.md
  - internal/acpfront/testdata/catalog/zeta-tools/SKILL.md
  - internal/acpfront/testdata/golden/desktop-delta-streaming.acp.golden
  - internal/acpfront/testdata/golden/desktop-delta-suppressed.acp.golden
  - internal/acpfront/testdata/golden/desktop-nodelta-reconcile.acp.golden
  - internal/acpfront/testdata/golden/desktop-terminal-signal.acp.golden
  - internal/acpfront/turnctx.go
  - internal/acpport/agent.go
  - internal/acpport/sink.go
  - internal/acpport/types.go
covered_digest: "v4:sha256:3639945f18a9e40d582bf87441abed53b1ba6f841b16c95355e6077966550dd6"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 8: Slash Catalog — available_commands_update — Verification Report

**Phase Goal:** the wrapper emits the `available_commands_update` catalog on `session/new`
(ZCode skills from SKILL.md frontmatter + the `/model` entry), within the client's 9 s
discovery scan budget, chat and discovery sessions alike; verified in the Orca composer.
**Verified:** 2026-11-23T11:17:28Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | CAT-01: ONE `session/update` with `available_commands_update` carrying `availableCommands` on every `session/new`, within the 9 s budget, chat and discovery alike | ✓ VERIFIED | `internal/acpport/agent.go` `NewSession` → `emitCommandCatalog` (synchronous, before the reply returns, lines 212/555-570); `sink.go` lines 50-69 map `NoteCatalog` → `acp.SessionAvailableCommandsUpdate{SessionUpdate: "available_commands_update", AvailableCommands}`. `TestSessionNewEmitsCommandCatalog` (catalog_emit_test.go) locks the count to exactly ONE frame per session/new and ONE per second session/new — PASS under `-race`. Budget: the emit rides inside session/new (no extra wait), and the VM field pass shows the composer menu populated — the omp UI renders items only when the frame arrived inside its own scan window (the alternative is the "No matching commands" empty state, per the dialect doc); the host-side card-argv runs measured +2.1 s…+5.5 s vs the 9 s budget (08-UAT.md, commit e18c527 message). Chat/discovery alike: the emission is unconditional on the session/new protocol path; `--session-dir` is accepted-and-ignored upstream (Phase 4), so both session kinds hit the identical emit. |
| 2 | CAT-02: the catalog lists the ZCode skills (user `~/.zcode/skills` + plugin caches), name/description from SKILL.md frontmatter, deterministic order, broken entries skipped without failing | ✓ VERIFIED | `internal/acpfront/catalog.go` `SkillItems` (glob `*/SKILL.md` per root, frontmatter parse `parseSkillManifest`, dir-name fallback, lowercase dedupe, `sort.Slice` by name, silent `continue` skips); `DefaultCatalogRoots` = user + `~/.agents/skills` compat + `~/.zcode/cli/plugins/cache/*/*/*/skills`. Tests: `TestSkillItems`, `TestSkillItemsDirNameFallback`, `TestDefaultCatalogRoots` (fixture roots incl. the broken/not-a-skill children) — PASS under `-race`. Source spot-check: `zcode-src/packages/services/src/skills/skillsService.ts#62-94` carries the same four roots, `normalizeSkillNameKey` lowercase dedupe and the readFrontmatter name-fallback — the scan conventions match. |
| 3 | CAT-03: the `/model` entry (name `model`, description naming the switch, `input.hint` for the selector) | ✓ VERIFIED | `internal/acpfront/front.go` `CommandCatalog` lines 725-729 append `{Name: "model", Description: "Switch the session model …", Hint: "<selector>"}`; `sink.go` lines 53-58 map `Hint` → `Input.Unstructured.Hint`. Emit test asserts `"model"` and `"hint":"\u003cselector\u003e"` in the client-visible frame — PASS. Golden frames byte-lock the item (`testdata/golden/desktop-nodelta-reconcile.acp.golden` line 1). |
| 4 | CAT-04: no `sessionCapabilities.delete` advertisement; `session/delete` answered by the SESS-02 teardown | ✓ VERIFIED | `internal/acpport/agent.go` `initializeResponse` lines 194-198 set `SessionCapabilities{Options: …}` only; the SDK field `Delete *SessionDeleteCapabilities json:"delete,omitempty"` (acp-go-sdk v0.20.0 `types_gen.go#8637+`) is never set → omitted on the wire. Lock: `TestSessionCapabilitiesAdvertisementStayedHonest` (sesscompat_test.go #565-609) bans `delete` (and fork/list/load/resume/close) in the parsed initialize reply — PASS. `DeleteSession` still delegates (agent.go #439-448, SESS-02). |
| 5 | CAT-05: field-verified — Orca's card composer autocomplete lists the skills/commands and `/model` for the wrapper card | ✓ VERIFIED | The VM GUI pass is recorded and its evidence inspected by this verification: `experiments/captures/v021-catalog-vm-menu.png` shows the Orca OMP Chat composer with `/` typed rendering `/model — Choose the model`, `/gsd-capture — Capture and scrub a field experiment`, `/gsd-manager [--analyze-deps] — Interactive command center …` — names, descriptions, argument-hints. `08-UAT.md` records the agent-executed pass (2026-11-23, standing VM-rendering delegation) with prerequisites verified. The user's real-machine look is recorded "awaiting" and is explicitly optional per REQUIREMENTS.md traceability ("the user's real-machine look optional"). |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Deferred Items

None — no failed truth matches a later phase (Phase 8 is the milestone's last phase).

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/acpfront/catalog.go` | skill/command scanner, deterministic, silent skips | ✓ VERIFIED | 262 lines; `SkillItems`, `CommandItems`, `DefaultCatalogRoots`, `DefaultCommandRoots`, `WorkspaceCatalogRoots`, `WorkspaceCommandRoots`; substantive, no stubs |
| `internal/acpfront/front.go` `CommandCatalog` | CatalogProvider callback assembling commands → skills → `/model` | ✓ VERIFIED | lines 701-732; ordered merge of both scans + the fixed model item |
| `internal/acpport/agent.go` `emitCommandCatalog` | ONE emission per session/new, never fails session/new | ✓ VERIFIED | lines 551-570; CatalogProvider type-assert opt-in, sink nil-safe, errors logged (Warn) not propagated |
| `internal/acpport/sink.go` NoteCatalog mapping | neutral items → wire `available_commands_update` | ✓ VERIFIED | lines 50-69; hint → `Input.Unstructured.Hint` only when non-empty |
| `internal/acpport/types.go` `NoteCatalog`/`CommandItem` | neutral vocabulary | ✓ VERIFIED | lines 192-218 |
| `cmd/zcode-acp/main.go` roots wiring | production roots injected | ✓ VERIFIED | lines 593-594: `WithCatalogRoots(DefaultCatalogRoots(home)…)`, `WithCommandRoots(DefaultCommandRoots(home)…)` |
| `docs/protocol/acp-client-dialect.md` | the wire truth + the shipped consequence note | ✓ VERIFIED | "The slash-command catalog surface" section + "Wrapper consequence — SHIPPED (v0.21.0 Phase 8)" |
| `README.md` | the card now lists commands; nothing to configure | ✓ VERIFIED | commit e18c527 adds README section "5. The slash-command autocomplete" |
| `internal/acpfront/catalog_internal_test.go` + `catalog_emit_test.go` | scanner + emitter locks | ✓ VERIFIED | 6 scanner + 3 emitter tests, all PASS at HEAD under `-race`; no skips |
| `internal/acpfront/testdata/golden/*.acp.golden` (4) | client-visible frame byte-lock on the ACP tee | ✓ VERIFIED | all 4 regenerated goldens carry the catalog frame as line 1; golden tests PASS (full suite green) |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| `cmd/zcode-acp/main.go` | `acpfront.Front` options | `WithCatalogRoots/WithCommandRoots(Default…Roots(home))` | WIRED | buildFront assembles both root sets when home resolves |
| `acpport.NewAgent` → `agentAdapter.NewSession` | `Front.CommandCatalog` | `CatalogProvider` interface type-assert | WIRED | `*Front` implements `CommandCatalog(ctx, sessionID) []acpport.CommandItem` |
| `emitCommandCatalog` | connection sink | `bindSink`/`UpdateSink` note publish | WIRED | NoteCatalog rides the same sink as the pump's chunk notes |
| `sink.go` | ACP SDK wire type | `acp.SessionAvailableCommandsUpdate` | WIRED | SDK union carries the discriminant + `AvailableCommands` |
| `Front.CommandCatalog` | disk | `filepath.Glob` + `os.ReadFile` over roots | WIRED | real scan, not a static list (only `/model` is a fixed wrapper capability, by design) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `SkillItems` | items | `Glob(root/*/SKILL.md)` + frontmatter read | Yes (live disk scan) | ✓ FLOWING |
| `CommandItems` | items | `Glob(root/*.md)` + frontmatter read | Yes (live disk scan) | ✓ FLOWING |
| `/model` item | fixed | wrapper capability constant | By design (not disk data) | ✓ FLOWING |
| golden frames | catalog line | recorded client-visible tee | Yes (byte-locked) | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Scanner contract (names/order/skip/fallback/roots) | `go test ./internal/acpfront/ -run 'TestSkillItems\|TestDefaultCatalogRoots\|TestCommandItems…' -race -v` | 6/6 PASS | ✓ PASS |
| ONE frame per session/new + model item + no-roots arm | `go test ./internal/acpfront/ -run 'TestSessionNewEmitsCommandCatalog\|TestSessionNewEmitsCommandRoots\|TestSessionNewCatalogNoRoots' -race -v` | 3/3 PASS | ✓ PASS |
| No delete advertisement (CAT-04 lock) | `go test ./internal/acpfront/ -run TestSessionCapabilitiesAdvertisementStayedHonest -race -v` | PASS | ✓ PASS |
| Whole workspace at HEAD | `go test -race ./...` | 10/10 packages ok (EXIT=0) | ✓ PASS |
| Pinned strict lint at HEAD | `mise exec -- golangci-lint run ./...` | 0 issues | ✓ PASS |
| Field evidence authenticity | read `experiments/captures/v021-catalog-vm-menu.png` | shows the claimed menu (`/model`, `/gsd-capture`, `/gsd-manager [--analyze-deps]` with hints) in the Orca OMP card | ✓ PASS |

A live re-run of the discovery argv was considered and deliberately NOT executed: a real
`session/new` spawns the user's zcode backend and a real account session (side effects);
the recorded VM pass + the tracer-harness emission tests cover the behavior deterministically.

### Probe Execution

No probes declared or found (`scripts/` holds only `vm-e2e.sh`; no `scripts/*/tests/probe-*.sh`).
The PLAN's binary-E2E arm is covered by the tracer-harness emission tests + the recorded VM pass.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| CAT-01 | 08-01 | ONE available_commands_update per session/new, within 9 s, chat+discovery | ✓ SATISFIED | Truth 1 |
| CAT-02 | 08-01 | skills listed (user + plugin caches), frontmatter names, deterministic, broken skipped | ✓ SATISFIED | Truth 2 |
| CAT-03 | 08-01 | the `/model` entry with selector hint | ✓ SATISFIED | Truth 3 |
| CAT-04 | 08-01 | no delete advertisement; SESS-02 answers session/delete | ✓ SATISFIED | Truth 4 |
| CAT-05 | 08-01 | field-verified in the Orca composer | ✓ SATISFIED | Truth 5 |

Orphaned requirements: none — REQUIREMENTS.md traceability maps exactly CAT-01..05 to Phase 8,
and the PLAN's verification table covers all five.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| catalog_internal_test.go | CAT-02 | 6 | 0 | No | Value/Behavioral (order, dedupe, fallback, skip) | SUFFICIENT |
| catalog_emit_test.go | CAT-01/03 | 3 | 0 | No | Behavioral (frame count, byte-contains, degraded arm) | SUFFICIENT |
| sesscompat_test.go (honesty lock) | CAT-04 | 1 | 0 | No | Value (parsed-JSON ban list) | SUFFICIENT |
| 4 regenerated .acp.golden files | CAT-01 | 4 suites | 0 | No (committed fixture bytes vs run output) | Value (byte-lock) | SUFFICIENT |

**Disabled tests on requirements:** 0. **Circular patterns:** 0 (goldens are committed
recordings compared read-only; the update flag is opt-in fixture maintenance).
**Insufficient assertions:** 0.

### Source Cross-Verification Spot-Checks

- orca clone (`/tmp/orca-src`): `src/main/acp/discovery/acp-at-rest-commands.ts#56` —
  `scanTimeoutMs: 9_000` (the 9 s budget claim CONFIRMED); `acp-session-discovery.ts#98-115` —
  the session-id-matched subscription filtering
  `update.sessionUpdate === 'available_commands_update'` (CONFIRMED).
- zcode clone (`/home/nil/DiskD/W/Djarvur/zcode-src`): `packages/services/src/skills/skillsService.ts#62-94` —
  the four skill roots, `normalizeSkillNameKey` lowercase dedupe, frontmatter name-fallback
  (CONFIRMED); `packages/services/src/commands/commandsService.ts#56-57,73-74` —
  `[".zcode","commands"]` + `[".agents","commands"]` at user and workspace levels (CONFIRMED).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | no TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER in any phase file; no stub returns; no skipped tests | — | — |

### Decision Coverage

Skipped — no CONTEXT.md in the phase directory (nothing to check).

### Deviations Noted (informational, no goal impact)

1. **The named corpus row `catalog-emission` does not exist as a scenario JSON** (PLAN Task 3
   letter). The intent — the client-visible `available_commands_update` frame byte-locked on the
   ACP tee over a scripted backend replay — is fully served by the 4 regenerated `.acp.golden`
   fixtures (catalog frame = line 1) plus the frame-count lock in
   `TestSessionNewEmitsCommandCatalog`. No requirement affected.
2. **Scope grew beyond the PLAN's skills-only scan**: the shipped catalog also lists commands
   (`~/.zcode/commands` + `~/.agents/commands` compat, item name = file stem) and the
   workspace root pairs, driven by the zcode-source verification (commandsService.ts layout,
   the runtime's stem invocation). This is a superset of the goal's "skills + /model" — both
   goal elements remain present and verified; README and the dialect doc document the shipped
   shape.
3. **The 139-item / +2.1 s…+5.5 s live measurement is recorded as prose** (08-UAT.md, commit
   e18c527 message) rather than a raw capture file. The VM screenshot independently proves the
   end-to-end outcome (the omp composer renders items only when the frame arrived inside its
   own scan window).
4. **The user's real-machine acceptance pass (08-UAT.md Test 2) is still "awaiting"** —
   explicitly optional per REQUIREMENTS.md traceability; the requirement's field-verification
   arm is the recorded agent-executed VM GUI pass.

### Human Verification Required

None required for the phase gate. The goal's "verified in the Orca composer" arm is satisfied
by the recorded VM GUI pass (agent-executed under the standing VM-rendering delegation,
evidence inspected). The optional user real-machine look (restart the chat session, see the
autocomplete list their real commands/skills) remains available to the user as an acceptance
nicety — it does not gate the phase.

### Gaps Summary

None. All five CAT requirements verify against real evidence: tests green at HEAD (9 catalog
tests + the CAT-04 honesty lock + the full 10-package suite under `-race`), the live catalog
measured (UAT prose + the GUI-pass screenshot whose rendering itself proves in-budget
arrival), and the GUI pass recorded (`experiments/captures/v021-catalog-vm-menu.png`,
08-UAT.md). The implementation is substantive and fully wired from the disk scan through the
front's CatalogProvider to the ACP sink's `available_commands_update` frame, with production
roots injected in `cmd/zcode-acp/main.go`.

---

_Verified: 2026-11-23T11:17:28Z_
_Verifier: Claude (gsd-verifier)_
