---
phase: 01-research-protocol-documentation
verified: 2026-11-19T09:00:09Z
status: human_needed
score: 5/5 ROADMAP success criteria verified; 27/28 plan must-have truths verified (1 UNCERTAIN routed to human judgment)
covered_files: [".gitignore", ".planning/phases/01-research-protocol-documentation/01-01-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-01-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-02-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-02-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-03-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-03-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-04-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-04-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-05-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-05-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-06-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-06-SUMMARY.md", "docs/protocol/README.md", "docs/protocol/acp-client-dialect.md", "docs/protocol/app-server-dialect.md", "docs/protocol/install-discovery.md", "docs/protocol/workspace-key.md", "docs/research/reference-adapters.md", "experiments/README.md", "experiments/app-server-probe.mjs", "experiments/captures/EXP-DISCOVERY-01.jsonl", "experiments/captures/EXP-SESSIONDIR-01.jsonl", "experiments/captures/EXP-SUBSCRIBE-01.jsonl", "experiments/captures/EXP-TURN-01.jsonl", "experiments/captures/EXP-WORKSPACEKEY-01.jsonl", "experiments/check-docs.sh", "experiments/scan-secrets.sh", "experiments/scrub-capture.mjs", "experiments/scrub-capture.test.mjs"]
covered_digest: "v4:sha256:f6e2487e7dc12efae110751848669361f833bc6a9331468ea70172635263031c"
behavior_unverified: 0
overrides_applied: 0
unverified_prohibition:
  requirement_id: DISC-02
  statement: "MUST NOT state client-dialect claims as current-build facts without the chunk-filename citation of the extract they were re-verified against"
  llm_judge_verdict: pass (NON-AUTHORITATIVE — judgment-tier, autonomous verify)
  evidence: "docs/protocol/acp-client-dialect.md read in full: every re-verified claim carries [V-SRC <chunk-filename>#<anchor>] (six chunk files cited in Sources & provenance); the three known-unknowns (settings-store location, live --version probe half, interruption semantics beyond the string contracts) are labeled [V-RPT], never presented as verified; the header names the build described and carries a build-drift warning"
  note: "human review recommended — the designed check per plan 01-04's own disposition is a human skim of the provenance tags"
mvp_goal_format_discrepancy:
  mode: mvp
  roadmap_goal_is_user_story: false
  verb: user-story.validate
  errors: ["Story must start with \"As a [user role],\" (role must be non-empty).", "Story must include \", I want to [capability],\" (capability must be non-empty).", "Story must include \", so that [outcome].\" (outcome must be non-empty)."]
  plan_reframe_validates: true
  action_requested: "Run /gsd:mvp-phase 1 to set a proper User Story goal in ROADMAP.md"
decision_coverage:
  honored: 6
  total: 6
  not_honored: []
deferred:
  - truth: "Turn-completion terminal envelopes observed live on the wire (the completion half of app-server-dialect.md section 5 is V-SRC-only by the D-05 degradation)"
    addressed_in: "Phase 6"
    evidence: "app-server-dialect.md section 5: 'source-level only until a credential-answering host exists (Phase 6)'; REQUIREMENTS.md AUTH-02 (Phase 6) covers interaction/requestCredentialCipher"
  - truth: "Known-unknowns: agentCmdOverrides settings-store location, live --version probe half, interruption semantics beyond the documented strings"
    addressed_in: "Phase 7"
    evidence: "acp-client-dialect.md Open items: all three labeled [V-RPT] 'Phase 7 live concern'; REQUIREMENTS.md SESS-03 (Phase 7) covers full dialect coverage live"
---

# Phase 1: Research & Protocol Documentation — Verification Report

**Phase Goal:** The ZCode app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics are documented from source and experiments, so every later phase builds on verified wire truth instead of assumptions.
**Verified:** 2026-11-19T09:00:09Z
**Status:** human_needed (no gaps; 4 human verification items below)
**Re-verification:** No — initial verification

## MVP Mode Note (process discrepancy — needs a human decision)

ROADMAP.md declares `**Mode:** mvp` for this phase, but the ROADMAP goal line is **not in user-story format** (`gsd_run query user-story.validate` → `valid: false`, all three slots missing). Per the MVP-mode guard this is surfaced as a discrepancy: run **`/gsd:mvp-phase 1`** to set a proper User Story goal, then re-run verification if canonical form matters to the record.

Verification was **not** refused wholesale, for auditability: (a) Step 2a makes the ROADMAP success criteria the non-negotiable contract in every mode, and those were verified in full below; (b) plan 01-01's objective contains an explicit reframe of the ROADMAP goal line into user-story form — "**As a** later-phase implementer on this repo, **I want to** the app-server dialect, … documented from source and experiments, **so that** every later phase builds on verified wire truth instead of assumptions." — which the same verb validates (`valid: true`, all slots extracted). The User Flow Coverage table below is derived from that verb-validated reframe, so the coverage section is not low-quality. No `passed` verdict is emitted regardless; the discrepancy routes to human action.

## User Flow Coverage

User story (plan 01-01 reframe, verb-validated): «As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Open the documentation index | One index names every deliverable doc + the single provenance legend | `docs/protocol/README.md` — deliverables table (5 DISC docs, each naming its DISC id), `## Provenance legend` defined exactly once (check-docs.sh check (a) green) | ✓ |
| Read the app-server dialect | Full JSONL event surface, subscribe verdict, terminal signals, per-claim provenance | `docs/protocol/app-server-dialect.md` — 10 sections (211 lines), 110-method map + 43 v4 strings with re-derivable commands/counts, E1 verdict table from `[V-EXP EXP-SUBSCRIBE-01]`, terminal-signal tables (turn-start live; completion honestly V-SRC-only), drift baseline first-class | ✓ |
| Read the client ACP dialect | omp identity, version floor, env quirks, method sets, string contracts | `docs/protocol/acp-client-dialect.md` — omp identity + alternatives ruled out, floor 17.0.5 / `resolvesSelectorsFrom:"18.1.9"`, PI_NO_PTY/agentCmdOverrides, startup + per-turn method sets, full 17-method range, verbatim Hb strings, chunk-filename citations | ✓ |
| Read the install discovery matrix | Locate commands per layout incl. auto-update effect on the revision string | `docs/protocol/install-discovery.md` — 5-row matrix (unpacked AppImage / AppImage file / deb / rpm / running-mount do-not-rely), deb artifact-inspected `3.18.2-10443`, revision formula `zcode-builtin:<rev>:<sha256(path.resolve(...))>` V-SRC + computed example, recommended discovery order | ✓ |
| Read the workspaceKey semantics | Source citation + recorded experiment | `docs/protocol/workspace-key.md` — verbatim 3.17.1 derivation quote (`workspaceIdentity?.trim() \|\| workspacePath`), nonEmptyString schema lines, `[V-EXP EXP-WORKSPACEKEY-01]` both create verdicts (path-as-key + arbitrary host string accepted) | ✓ |
| Read the reference comparison | Coverage, adopt, avoid under docs/research/ | `docs/research/reference-adapters.md` — 12-row coverage map at pinned tag v0.89.0, golden source+test files, Avoid list, Apache-2.0 attribution, weaker adapters, Start-Plan headless note | ✓ |
| Outcome: later phases build on verified wire truth | Claims carry resolving provenance; gates enforce it | All 105 `[V-EXP …]` citations resolve to the 5 scrubbed captures (headers intact, builtinRevision 44 pinned); `bash experiments/check-docs.sh --phase-gate` exit 0; later-phase consumption stated per doc (Phase 3 §1/2/6, Phase 4 §4–5, Phase 5 §3, Phase 6 §8; REQUIREMENTS.md DISC-01..05 marked Complete) | ✓ |

## Goal Achievement

### Observable Truths

ROADMAP success criteria (the contract):

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|--------------------|--------|----------|
| 1 | `docs/` protocol documentation describes the app-server JSONL event surface — incl. `session/subscribe`/event-stream availability and turn-completion terminal signals — every claim labeled V-SRC or V-EXP against the shipped 3.18.2 binary | ✓ VERIFIED | `docs/protocol/app-server-dialect.md`: §4 subscribe verdict table (session/subscribe ACCEPTED, session/events ACCEPTED, v4/conversation/subscribe REFUSED −32603) from the capture's per-method verdict lines (verified in `EXP-SUBSCRIBE-01.jsonl`: 2 accepted + 1 refused); §5 terminal signals (turn-start live `[V-EXP EXP-TURN-01]`; completion half V-SRC-only with the D-05 honesty statement — the deliberately ledgered degradation, labeled accurately per the honest-evidence contract). Every claim line carries a tag; tag grammar machine-enforced by check-docs.sh check (c) |
| 2 | A dialect document names the chosen preset-agent identity with version floor and env quirks, and enumerates the load-bearing method set at card startup/per turn and the full ACP method range | ✓ VERIFIED | `docs/protocol/acp-client-dialect.md`: omp identity + ruled-out alternatives with chunk citations; floor `17.0.5` (in doc and capture ×3), `resolvesSelectorsFrom:"18.1.9"`; `PI_NO_PTY=1`, `PI_CODING_AGENT_DIR`, full agentCmdOverrides semantics; startup sequence (initialize → session/new → authenticate? → load/resume?) vs per-turn set kept distinct with overlap cross-reference; 17-row full method range table |
| 3 | A discovery matrix documents how to locate a ZCode desktop install across AppImage / deb / unpacked layouts, including what auto-update does to the builtin revision string | ✓ VERIFIED | `docs/protocol/install-discovery.md`: matrix rows 1/2/3 V-EXP-backed (16 `[V-EXP EXP-DISCOVERY-01]` citations; capture has 4 `dpkg-deb` hits, pristine/identical ×2), row 4 deb-generalized with header-verified version scheme, row 5 do-not-rely (×3 in doc); Auto-update section: updater pending-cache mechanics observed, runtime tree accretion 28→30→31→43→44, revision formula V-SRC with bundle literals + computed example `zcode-builtin:44:e9c0674a…` (present in capture); live install/apply step honestly V-RPT |
| 4 | `workspaceKey` derivation semantics documented with both a source citation and a recorded experiment | ✓ VERIFIED | `docs/protocol/workspace-key.md`: V-SRC citations to 3.17.1 `index.ts:2762/:2784` (nonEmptyString required/optional), `official-mcp-auth.ts:149` (env), `zcode-protocol-entrypoint.ts:219-236` (verbatim convention quote); `[V-EXP EXP-WORKSPACEKEY-01]` cited ×13, capture verified to contain both verdicts (`probe-arbitrary-host-key-01` ×7) with header intact |
| 5 | A reference-implementation comparison under `docs/research/` records what the TypeScript zcode-acp project covers (elicitation, slash commands, resume/fork, attachments), what to adopt, and what to avoid | ✓ VERIFIED | `docs/research/reference-adapters.md`: 12-row coverage map incl. rows 6 (elicitation with permission fallback), 8 (slash commands), 9 (resume/fork), 10 (attachments) — each with V-SRC anchors pinned to tag v0.89.0; `## Adopt` (5 golden source files + 3 golden test files), `## Avoid` (6 items with reasons), `## Attribution` (Apache-2.0 §4(b)/(d) obligations, no-NOTICE finding); weaker-adapters table; Start-Plan headless limitation feeding DOCS-03 |

Plan-level must-have truths (28 across the six plans): **27 VERIFIED, 1 UNCERTAIN** — the one exception is 01-02 truth 3 ("Exactly one real turn is spent… the ledger says so; no committed script can mint API keys"): the no-minting half is machine-verified (`api_keys` absent from all harness scripts — check-docs.sh check (g) + independent grep ×0), but the actual spend was **3 tiny sends issued, 0 completed** (first-token latency exceeded the quiet window twice; the mid-turn `requestCredentialCipher` obligation — needing keyring material out of Phase-1 scope — blocked completion, and a 4th run was correctly refused). The plan's own D-05 degradation branch was executed and ledgered honestly (`experiments/README.md:183`), the docs label it accurately, and plan 01-02 itself marked exactly this item `human_judgment: true`. Whether the "3 sends / 0 completed" framing satisfies the intent of "exactly one real turn" is a human call → Human Verification item 3.

**Score:** 5/5 ROADMAP SCs; 27/28 plan truths (0 present-but-behavior-unverified)

### Required Artifacts

All 20 artifacts declared across the six plans verified at Level 1–3. Selected highlights:

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `docs/protocol/README.md` | Index + single provenance legend | ✓ VERIFIED | Legend defined exactly once (check (a)); indexes all 5 DISC docs (+ itself = the six-doc set) |
| `docs/protocol/app-server-dialect.md` | DISC-01 deliverable, 9 sections + Sources | ✓ VERIFIED | All 10 headings present in the exact form check-docs.sh canonicalizes (incl. `## 9. 3.17 vs 3.18.2 drift baseline`) |
| `docs/protocol/acp-client-dialect.md` | DISC-02 deliverable, 10 sections | ✓ VERIFIED | All headings; 43 V-SRC chunk citations; 11 V-RPT; 5 V-EXP resolving |
| `docs/protocol/install-discovery.md` | DISC-03 deliverable, 6 sections | ✓ VERIFIED | Matrix + entrypoint + URL-handler clash + auto-update/revision + discovery order + provenance |
| `docs/protocol/workspace-key.md` | DISC-04 deliverable | ✓ VERIFIED | Wire schema / derivation convention / experiment record / host-freedom note / sources |
| `docs/research/reference-adapters.md` | DISC-05 deliverable, 7 sections | ✓ VERIFIED | Coverage map / Adopt / Avoid / Attribution / Weaker adapters / Limitations / Sources |
| `experiments/check-docs.sh` | Conformance + phase gate | ✓ VERIFIED | exit 0 both modes; checks (a)–(g) incl. capture-header continuity and no-mint gate |
| `experiments/scan-secrets.sh` | Secret-shape gate | ✓ VERIFIED | exit 0; value-shape patterns; .planning excluded |
| `experiments/scrub-capture.mjs` (+ test) | Canonical scrubber + tests | ✓ VERIFIED | 12/12 node --test; end-to-end idempotency proven on committed capture (byte-identical re-scrub) |
| `experiments/app-server-probe.mjs` | Flow harness | ✓ VERIFIED | node --check clean; probe/workspacekey/subscribe/turn flows present; quiet-window end mode |
| `experiments/README.md` | Ops guide + quota ledger | ✓ VERIFIED | E1–E5 shapes match implemented flags; never-in-CI warning; ledger line states actual spend |
| `experiments/captures/EXP-*.jsonl` (5) | Scrubbed captures, headers intact | ✓ VERIFIED | All 5 headers carry `{"capture":"<id>",…,"builtinRevision":"44"}`; content spot-checks match every doc claim (verdict lines, degradation record, revision example, session-dir verdicts) |
| `.gitignore` | `experiments/raw/` defense | ✓ VERIFIED | Line 37 |

Mechanical-check notes (audited against reality, not waved through):
- `verify.artifacts` on 01-03 reports one miss: plan `contains: "## 3.17 vs 3.18.2 drift baseline"` vs actual heading `## 9. 3.17 vs 3.18.2 drift baseline` (numbered). The section exists first-class (line 184, full drift content), and the phase's own conformance gate requires and passes exactly the numbered form. Ruled: content requirement satisfied; pattern-literalism false positive, not a stub (the doc is 211 lines, wired, gate-green).
- `verify.key-links` reports 3 misses of pattern `docs/protocol/README` in workspace-key.md / app-server-dialect.md / install-discovery.md. All three reference the legend via sibling-relative links (`[README.md](README.md)`, lines 7/8/8 respectively), and check-docs.sh check (a) enforces "legend defined once, others reference the README" — the link is wired. Ruled: WIRED; grep-pattern false negative.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| workspace-key.md | EXP-WORKSPACEKEY-01.jsonl | `[V-EXP EXP-WORKSPACEKEY-01]` resolves | WIRED | 13 citations; capture exists with intact header |
| app-server-probe.mjs | captures | scrub-first pipeline | WIRED | scrub-capture.mjs is the only writer into captures/; idempotency proven end-to-end |
| app-server-dialect.md | EXP-SUBSCRIBE-01 / EXP-TURN-01 | V-EXP tags | WIRED | 26 + 38 citations respectively, all resolving |
| app-server-dialect.md | docs/protocol/README.md | legend referenced | WIRED | Relative `[README.md](README.md)` link at line 8 (mechanical pattern-miss documented above) |
| acp-client-dialect.md | EXP-SESSIONDIR-01 | V-EXP tag | WIRED | 5 citations resolving; capture holds both open-item verdicts + `17.0.5` ×3 |
| install-discovery.md | EXP-DISCOVERY-01 | V-EXP tag | WIRED | 16 citations; capture holds dpkg-deb, pristine, computed revision example |
| install-discovery.md | docs/protocol/README.md | legend referenced | WIRED | Relative link at line 8 (pattern-miss documented above) |
| reference-adapters.md | docs/protocol/README.md | legend referenced | WIRED | `../protocol/README.md` at line 8 |
| reference-adapters.md | .planning/research/REFERENCE-ADAPTERS.md | `credential-store` claim verification | WIRED | Golden file named ×3; drift row recorded (no key creation in v0.89.0) |
| README.md (protocol) | all five DISC docs | deliverables index | WIRED | Table rows link each doc with its DISC id |

### Data-Flow Trace (Level 4)

Documentation phase: the analog of "rendered value → real data source" is "doc claim → committed evidence".

| Artifact | Claim source | Traced to | Real data | Status |
| -------- | ------------ | --------- | --------- | ------ |
| app-server-dialect.md §4/§5 | V-EXP claims | EXP-SUBSCRIBE-01 / EXP-TURN-01 verdict + wire lines | Live 3.18.2 runs recorded via scrub-first pipeline | ✓ FLOWING |
| workspace-key.md | V-EXP + V-SRC | EXP-WORKSPACEKEY-01 + 3.17.1 clone citations | Live run + version-pinned source | ✓ FLOWING |
| acp-client-dialect.md | V-SRC(chunk) + V-EXP | 6 named chunk files (fresh extract) + EXP-SESSIONDIR-01 | Static chunk reads recorded as literals+anchors (tier-compliant) | ✓ FLOWING |
| install-discovery.md | V-EXP + V-SRC | EXP-DISCOVERY-01 (7 rows) + bundle literals | Read-only live enumeration + artifact inspection | ✓ FLOWING |
| reference-adapters.md | V-SRC(upstream) | Tag-pinned v0.89.0 clone (scratch, outside repo — by design) | Live study; nothing upstream committed (`git ls-files` gate exit=1 verified by executor; no `*credential-store*` tracked file exists) | ✓ FLOWING |
| No static/hardcoded data paths found | — | — | — | none flagged |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Docs conformance gate | `bash experiments/check-docs.sh` | exit 0, no CHECK FAIL | ✓ PASS |
| All-six phase gate | `bash experiments/check-docs.sh --phase-gate` | exit 0, 0 PENDING | ✓ PASS |
| Secret-shape gate | `bash experiments/scan-secrets.sh` | exit 0, no SECRET SHAPE | ✓ PASS |
| Scrubber unit behavior | `node --test experiments/` | 12 pass / 0 fail | ✓ PASS |
| Scrubber end-to-end idempotency on committed capture | `node experiments/scrub-capture.mjs experiments/captures/EXP-WORKSPACEKEY-01.jsonl` + `cmp` | byte-identical | ✓ PASS |
| Harness syntax + flows | `node --check experiments/app-server-probe.mjs`; grep subscribe/turn flows | clean; flows present | ✓ PASS |
| No key-minting endpoint | grep `api_keys` in experiments scripts | only check-docs.sh's own gate text (self-excluded by design) | ✓ PASS |
| V-EXP resolution (independent of gate) | 105 bracketed citations vs 5 capture files | all resolve | ✓ PASS |
| Commit ancestry (15 hashes from summaries) | `verify.commits` | all_valid: true | ✓ PASS |

The E1/E2/E3/E4/E5 live runs themselves are not re-executable by the verifier without re-spending quota and re-touching live account state; their evidence is the committed captures, which were content-verified against every doc claim (verdict lines, degradation record, turn-start kinds, session-dir verdicts, dpkg/pristine/revision rows).

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| docs conformance | `bash experiments/check-docs.sh` | exit 0 | PASS |
| phase gate (strict, all six docs) | `bash experiments/check-docs.sh --phase-gate` | exit 0 | PASS |
| secret scan | `bash experiments/scan-secrets.sh` | exit 0 | PASS |
| unit tests | `node --test experiments/` | 12/12 pass | PASS |

### Requirements Coverage

All five phase requirement IDs from REQUIREMENTS.md are claimed by exactly one plan each and are satisfied. No orphaned requirements (REQUIREMENTS.md maps only DISC-01..05 to Phase 1; all appear in plan frontmatter).

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DISC-01 | 01-02 (evidence) + 01-03 (doc) | Event surface documented from source, V-SRC/V-EXP labeled | ✓ SATISFIED | app-server-dialect.md complete; E1 verdicts + E2 degradation captured; drift baseline; `session/delete` absence ×0 recorded |
| DISC-02 | 01-04 | Client dialect: identity, floor, method sets, full range; spot-check items answered | ✓ SATISFIED | acp-client-dialect.md (omp, 17.0.5/18.1.9, agentCmdOverrides, Hb strings); `--session-dir` format + modelDiscovery kind session answered via EXP-SESSIONDIR-01 |
| DISC-03 | 01-05 | Desktop discovery matrix incl. auto-update effect on revision | ✓ SATISFIED | install-discovery.md 5-layout matrix; revision formula sharpened V-SRC; deb row artifact-inspected |
| DISC-04 | 01-01 | workspaceKey semantics: source + experiment | ✓ SATISFIED | workspace-key.md + EXP-WORKSPACEKEY-01 |
| DISC-05 | 01-06 | Reference implementation compared; findings in docs/research/ | ✓ SATISFIED | reference-adapters.md; live v0.89.0 study; drift recorded; attribution settled |

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts (6/6 honored, 0 not honored; gate `check.decision-coverage-verify` exit 0, non-blocking by design).

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| experiments/scrub-capture.test.mjs | D-02 scrub rules (pipeline integrity) | 12 | 0 | 0 (writeFileSync writes fixture *inputs* to a mkdtemp dir, not SUT-generated expected values) | Value/Behavioral (placeholder determinism, byte-identity, whole-value redaction) | SUFFICIENT |

**Disabled tests on requirements:** 0. **Circular patterns:** 0. **Insufficient assertions:** 0. Doc-conformance "tests" are the check-docs.sh checks themselves (structure, tag grammar, V-EXP resolution, header continuity, path shapes, no-mint) — all exercised above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) across docs/ + experiments/ | none found | debt-marker gate clean |
| docs/protocol/*.md, experiments/* (7 files) | various | lowercase "placeholder" | ℹ️ Info | prose/code describing the scrubber's placeholder conventions and the path-note rule — domain language, not stubs |
| — | — | empty implementations / console-log-only / hardcoded empty data | none found | — |

Post-phase review advisories (01-REVIEW.md: 0 critical, 4 warning, 4 info) are dispositioned in 01-REVIEW-DISPOSITION.md with explicit batch targets (Phase 2 CI gate adoption, next doc touch, Phase 3 harness work). They are quality advisories on working deliverables — none blocks goal achievement; recorded here so verify-work sees them.

## Human Verification Required

### 1. MVP goal format — run /gsd:mvp-phase 1 (process decision)

**Test:** Run `/gsd:mvp-phase 1` to put the ROADMAP Phase 1 goal into canonical User Story form, then decide whether re-verification against the reformatted goal is wanted.
**Expected:** ROADMAP goal matches `As a …, I want to …, so that ….` (the plan 01-01 reframe already validates if you prefer to paste it).
**Why human:** The MVP-mode guard requires surfacing this discrepancy for a human decision; the verifier must not reformat the roadmap itself.

### 2. Provenance spot-check of V-SRC citations against the machine-local sources

**Test:** Pick ~5 load-bearing claims (e.g. `index.ts:2762` nonEmptyString; the `Reasoning level is required` ×1 bundle count; `cmp` pristine result; omp `minimumVersions:["17.0.5"]` chunk anchor; upstream `project_token_key_missing` in the v0.89.0 clone) and re-run the recorded commands against the local 3.18.2 bundle, 3.17.1 clone, client AppImage extract, and upstream clone.
**Expected:** Each re-derivation reproduces the documented anchor/count.
**Why human:** The cited sources (shipped bundle, source clone, client build, upstream repo) are machine-local by design and never committed; the repo cannot self-verify them. The plans' own disposition designates this human skim as the designed check.

### 3. Quota accounting framing (01-02 truth: "exactly one real turn")

**Test:** Read `experiments/README.md` run notes + the EXP-TURN-01 degradation record; confirm "3 tiny sends issued, 0 completed" is an acceptable honest accounting against the plan's "exactly one real turn" (the deviation was forced by the mid-turn `requestCredentialCipher` obligation and halted at 3 runs by self-imposed discipline).
**Expected:** Human accepts the framing (or records an override / a follow-up action).
**Why human:** Judgment call the plan itself flagged `human_judgment: true`; the letter of the truth did not hold, the intent (bounded spend, honest ledger, no minting) is machine-verified.

### 4. Judgment-tier prohibition (DISC-02 citation honesty) — human skim

**Test:** Skim `docs/protocol/acp-client-dialect.md` provenance tags: every current-build claim should cite its chunk filename; unconfirmed claims should stay V-RPT.
**Expected:** Confirmed (LLM-judge pre-verdict: pass — 43 chunk citations, 11 V-RPT, build drift warning in header; NON-AUTHORITATIVE).
**Why human:** Citation honesty cannot be reduced to a mechanical test (plan 01-04's own disposition); autonomous verify must flag, never silently pass.

## Gaps Summary

None. Every ROADMAP success criterion and 27 of 28 plan must-have truths are verified with machine-checkable evidence; the mechanical-verb misses (1 artifact contains-pattern, 3 link patterns) were audited against reality and are false positives of pattern literalism, documented above with the wiring evidence. The remaining items are human judgments (goal format, provenance skim, quota framing, prohibition skim), so the status is `human_needed`, not `passed`.

---

_Verified: 2026-11-19T09:00:09Z_
_Verifier: Claude (gsd-verifier)_
