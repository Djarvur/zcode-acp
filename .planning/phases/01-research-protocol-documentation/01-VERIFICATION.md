---
phase: 01-research-protocol-documentation
verified: 2026-11-20T11:04:00Z
status: passed
score: 5/5 ROADMAP success criteria verified; 28/28 plan must-have truths verified (1 via recorded human-acceptance override)
covered_files: [".gitignore", ".planning/phases/01-research-protocol-documentation/01-01-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-01-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-02-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-02-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-03-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-03-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-04-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-04-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-05-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-05-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-06-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-06-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW-DISPOSITION.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW-FIX.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW.md", ".planning/phases/01-research-protocol-documentation/01-SECURITY.md", ".planning/phases/01-research-protocol-documentation/01-UAT.md", ".planning/phases/01-research-protocol-documentation/01-VALIDATION.md", "docs/protocol/README.md", "docs/protocol/acp-client-dialect.md", "docs/protocol/app-server-dialect.md", "docs/protocol/install-discovery.md", "docs/protocol/workspace-key.md", "docs/research/reference-adapters.md", "experiments/README.md", "experiments/app-server-probe.mjs", "experiments/captures/EXP-DISCOVERY-01.jsonl", "experiments/captures/EXP-ORCASRC-01.jsonl", "experiments/captures/EXP-SESSIONDIR-01.jsonl", "experiments/captures/EXP-SUBSCRIBE-01.jsonl", "experiments/captures/EXP-TURN-01.jsonl", "experiments/captures/EXP-WORKSPACEKEY-01.jsonl", "experiments/check-docs.sh", "experiments/scan-secrets.sh", "experiments/scrub-capture.mjs", "experiments/scrub-capture.test.mjs"]
covered_digest: "v4:sha256:ec0940824c254666dd64c6e4c37c1494fd1e9dc2c9d9306ee700ef1a43712cab"
behavior_unverified: 0
overrides_applied: 1
overrides:
  - must_have: "Exactly one real turn is spent on the account for this phase; the experiments README quota ledger says so; no committed script can mint API keys"
    reason: "The no-minting half is machine-verified (api_keys absent from harness scripts — re-grepped this run: 0 hits in probe + scan-secrets; check-docs.sh mint gate at lines 145–148 green in this run's exit-0 run). The letter of the spend half did not hold (3 tiny sends issued, 0 completed: first-token latency outran the quiet window twice; mid-turn requestCredentialCipher — keyring material out of Phase-1 scope — blocked completion; 4th run correctly refused). The plan's D-05 degradation branch was executed and ledgered honestly (ledger line re-read this run at experiments/README.md:189: 'real turns spent: 3 tiny sends issued, 0 completed'); the human accepted the framing per UAT item 3 (2026-11-19: spend practically zero, 3 truncated requests, no model replies)."
    accepted_by: "user (recorded in 01-UAT.md item 3)"
    accepted_at: "2026-11-19T11:34:57Z"
prohibitions:
  - requirement_id: DISC-02
    statement: "MUST NOT state client-dialect claims as current-build facts without the chunk-filename citation of the extract they were re-verified against; build drift stays visible"
    status: verified
    verification: judgment
    resolution: "Resolved with stronger evidence than the designed human skim: the whole doc was cross-verified against the public client source (github.com/stablyai/orca, MIT) at tag v1.4.262 @13619bc1 — claim groups dual-cited as [V-SRC <chunk>#<anchor> ... repo src/<path>#<line> @v1.4.262], 3 chunk-only interpretations corrected and flagged in place (commit ed897a8, capture EXP-ORCASRC-01), 3 genuinely-unresolved items stay V-RPT (x11 in doc, re-counted this run), build-drift warning in header names both the installed build and the citation tag. Human-accepted per UAT item 4. Regression re-checked this run (doc byte-identical since the prior pass): V-RPT count 11, [V-EXP EXP-ORCASRC-01] tag present x1 and gate-covered, header drift warning at line 13, header names the same three known-unknowns (settings-store path, live end-to-end --version, interruption semantics) as the three V-RPT bullets in the Open items section."
re_verification:
  previous_status: passed
  previous_score: "5/5 ROADMAP success criteria; 28/28 plan truths (1 override)"
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Staleness refresh (#4682), not a gap-closure round: the prior verdict was passed and stays passed. Exactly one covered file changed since the prior report (commit f7de0a5 held digest v4:5202): experiments/scan-secrets.sh, +13 lines in a49486f ('fix(03): allowlist the synthetic fixture marker in the secret scan (CI unblock)') — an exact-string allowlist for the Phase-3 golden-corpus fixture marker. That file was re-verified in full (Levels 1–4 + behavioral: necessary-change proof, synthetic-shape battery through the actual pattern+allowlist pipeline, gate re-run, anti-pattern scan). All other covered files are byte-identical to the prior verified state (git diff f7de0a5..HEAD touches no other covered path) and got existence + sanity regression: all four phase-surface gates re-run green (check-docs.sh, --phase-gate, scan-secrets.sh, node --test 22/22), all 6 capture headers intact, all [V-EXP] citations resolve independently of the gate, prohibition/override/deferred/advisory states re-read at source. Fingerprint recomputed over the current 37-file covered set. Mechanical verbs reproduce the same three documented pattern-literalism false positives as the prior pass (re-audited below). The decision-coverage gate now RUNS (this pass passed the phase-scoped 01-CONTEXT.md explicitly): 6/6 decisions honored. Phase 1 has no Go code and no Go-based must-haves; the dirty internal/zcode/conn_test.go in the worktree is the Phase-5 parallel session's in-flight work and is out of scope for this phase."
deferred:
  - truth: "Turn-completion terminal envelopes observed live on the wire (the completion half of app-server-dialect.md section 5 is V-SRC-only by the D-05 degradation)"
    addressed_in: "Phase 6"
    evidence: "app-server-dialect.md:139 re-read this run: 'Explicit finish events (turn.completed/turn.failed): source-level only until a credential-answering host exists (Phase 6) — V-SRC-only, limitation as stated'; REQUIREMENTS.md AUTH-02 (Phase 6, Pending) covers interaction/requestCredentialCipher (re-checked this run at REQUIREMENTS.md:48,134)"
  - truth: "Known-unknowns: agentCmdOverrides settings-store location, live --version probe half against the real wrapper, interruption semantics beyond the documented strings"
    addressed_in: "Phase 7"
    evidence: "acp-client-dialect.md Open items re-read this run: all three unresolved bullets labeled V-RPT with 'Phase 7 live concern'/'Phase 7 plan' markers; REQUIREMENTS.md SESS-03 (Phase 7) covers full dialect coverage live (REQUIREMENTS.md:56,146)"
advisory:
  - finding: "TOKEN_RX in experiments/scrub-capture.mjs matches bare-quote keys only — an escaped-form \\\"access_token\\\":\\\"<20+ chars>\\\" inside the probe envelope is neither scrubbed nor caught by scan-secrets.sh pattern (c) (whose required unescaped opening quote also fails on the escaped backslash-quote sequence). JWT/credential/UUID rules are shape-based and safe."
    category: security
    reason: "Carried forward from the prior report: recorded by the fix batch itself as a follow-up lead in 01-REVIEW-FIX.md ('Notes for follow-up'), batched for the next review pass; no test exercises the escaped-form token shape, so it is flagged rather than counted against any must-have. Exposure is narrow: bare 20+ char token VALUES in escaped envelope form; the committed captures contain zero user_info/token payloads (swept by 01-REVIEW.md). Re-confirmed this run: scrub-capture.mjs and the doc set are byte-identical since the prior pass (only scan-secrets.sh changed), so the gap is unchanged; the new scan-secrets allowlist does not widen it (fixed-string drop of one self-declaring marker line only — proven by the synthetic battery)."
    evidence_status: "recorded — 01-REVIEW-FIX.md follow-up note; unchanged since prior pass (git diff f7de0a5..HEAD excludes the scrubber); synthetic battery this run confirms non-marker token shapes still fail the gate"
---

# Phase 1: Research & Protocol Documentation — Verification Report

**Phase Goal:** As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.
**Verified:** 2026-11-20T11:04:00Z
**Status:** passed
**Re-verification:** Yes — staleness refresh (#4682): the prior passed report's covered-input digest went stale (one covered file changed); verdict re-derived against the current tree

## Re-verification Scope

Changed inputs since the previous report (commit f7de0a5 held digest `v4:5202`): exactly **one** covered file — `experiments/scan-secrets.sh`, +13 lines in a49486f ("fix(03): allowlist the synthetic fixture marker in the secret scan (CI unblock)"). The change adds an exact-string allowlist (`grep -vF 'enc:v2:os:fix1-synthetic-not-a-real-credential'`) applied before classification; the three secret-shape patterns themselves are untouched. Per re-verification rules: the changed file got full Levels 1–4 + behavioral verification (see Behavioral Spot-Checks — including proof the change was *necessary* and proof the gate's catching power is *intact*); every other covered file is byte-identical to the prior verified state (git diff f7de0a5..HEAD touches no other covered path) and got existence + sanity regression — all green. The verdict is re-derived from this run's own evidence, not inherited.

## MVP Mode Guard

`**Mode:** mvp` with the goal in validated user-story form: `user-story.validate` on the current ROADMAP goal returns **valid: true** (re-run this session, exit 0). No refusal path applies.

## User Flow Coverage

User story (ROADMAP goal, verb-validated this run): «As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Open the documentation index | One index names every deliverable doc + the single provenance legend | `docs/protocol/README.md` — deliverables table naming DISC-01..05 (re-grepped this run: all five ids present), `## Provenance legend` defined exactly once (check-docs.sh check (a) green this run) | ✓ |
| Read the app-server dialect | Full JSONL event surface, subscribe verdict, terminal signals, per-claim provenance | `docs/protocol/app-server-dialect.md` (unchanged, regression-green): 10 sections incl. `## 9. 3.17 vs 3.18.2 drift baseline` (line 184), E1 verdict table from `[V-EXP EXP-SUBSCRIBE-01]` (26 tag occurrences, all resolving), terminal-signal tables (turn-start live; completion honestly V-SRC-only with the D-05 statement re-read this run at line 139), drift baseline first-class | ✓ |
| Read the client ACP dialect | omp identity, version floor, env quirks, method sets, string contracts — source-corroborated | `docs/protocol/acp-client-dialect.md` (unchanged, regression-green): omp identity + alternatives ruled out with dual citations; floor 17.0.5 (doc ×2 + EXP-SESSIONDIR-01 capture ×3); header↔body consistency re-verified this run — header names "the three remaining known-unknowns (settings-store persistence path, live end-to-end `--version`, interruption semantics)" (lines 10–12, phrase wraps across the line break) and the Open items section carries exactly those same three V-RPT bullets (lines 477–492); `[V-EXP EXP-ORCASRC-01]` tag ×1 gate-covered; build-drift warning in header (line 13); V-RPT ×11 | ✓ |
| Read the install discovery matrix | Locate commands per layout incl. auto-update effect on the revision string | `docs/protocol/install-discovery.md` unchanged: 5-row matrix, 23 `[V-EXP EXP-DISCOVERY-01]` tag occurrences all resolving (capture header intact), revision formula V-SRC, recommended discovery order | ✓ |
| Read the workspaceKey semantics | Source citation + recorded experiment | `docs/protocol/workspace-key.md` unchanged: verbatim 3.17.1 derivation quotes, 13 `[V-EXP EXP-WORKSPACEKEY-01]` tag occurrences all resolving, capture header intact | ✓ |
| Read the reference comparison | Coverage, adopt, avoid under docs/research/ | `docs/research/reference-adapters.md` unchanged: 12-row coverage map at v0.89.0, golden source+test files, Avoid list, `## Attribution` present | ✓ |
| Outcome: later phases build on verified wire truth | Claims carry resolving provenance; gates enforce it | All `[V-EXP …]` citations resolve to the 6 scrubbed captures (independent re-check this run, all 6 headers carry `{"capture":"<id>"}`); `bash experiments/check-docs.sh --phase-gate` exit 0 re-run by this verifier; `scan-secrets.sh` exit 0 re-run (now with the Phase-3 fixture allowlist); `node --test experiments/` 22/22 re-run; later-phase consumption stated per doc; REQUIREMENTS.md DISC-01..05 all marked Complete | ✓ |

## Goal Achievement

### Observable Truths

ROADMAP success criteria (the contract):

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|--------------------|--------|----------|
| 1 | `docs/` protocol documentation describes the app-server JSONL event surface — incl. `session/subscribe`/event-stream availability and turn-completion terminal signals — every claim labeled V-SRC or V-EXP against the shipped 3.18.2 binary | ✓ VERIFIED (regression — artifact unchanged) | `docs/protocol/app-server-dialect.md`: §4 subscribe verdict table (session/subscribe ACCEPTED, session/events ACCEPTED, v4/conversation/subscribe REFUSED −32603; EXP-SUBSCRIBE-01 citations all resolving — 26 tag occurrences re-counted this run); §5 terminal signals (turn-start live `[V-EXP EXP-TURN-01]` ×38 occurrences; completion honestly V-SRC-only with the D-05 statement at line 139). Tag grammar machine-enforced by check-docs.sh check (c), green this run |
| 2 | A dialect document names the chosen preset-agent identity with version floor and env quirks, and enumerates the load-bearing method set at card startup/per turn and the full ACP method range | ✓ VERIFIED (regression — artifact unchanged) | `docs/protocol/acp-client-dialect.md`: omp identity + ruled-out alternatives, dual-cited chunk + repo @v1.4.262; floor `17.0.5`; `PI_NO_PTY=1`/`PI_CODING_AGENT_DIR`/`OMP_CODING_AGENT_DIR`; startup vs per-turn sets distinct; full range table 19 rows incl. `session/close` + `elicitation/complete` with repo citations; Hb string contracts with matcher precision; header↔Open-items count consistency and single-claim-strength discipline re-verified this run (IN-01/IN-02 regression green); EXP-ORCASRC-01 in gate-covered V-EXP tag form |
| 3 | A discovery matrix documents how to locate a ZCode desktop install across AppImage / deb / unpacked layouts, including what auto-update does to the builtin revision string | ✓ VERIFIED (regression — artifact unchanged) | `docs/protocol/install-discovery.md`: 5-row matrix with resolving `[V-EXP EXP-DISCOVERY-01]` citations (23 tag occurrences); auto-update section with runtime-tree accretion + revision formula V-SRC; live install/apply step honestly V-RPT |
| 4 | `workspaceKey` derivation semantics documented with both a source citation and a recorded experiment | ✓ VERIFIED (regression — artifact unchanged) | `docs/protocol/workspace-key.md`: V-SRC citations to 3.17.1 sources; `[V-EXP EXP-WORKSPACEKEY-01]` ×13 occurrences resolving, capture header intact |
| 5 | A reference-implementation comparison under `docs/research/` records what the TypeScript zcode-acp project covers (elicitation, slash commands, resume/fork, attachments), what to adopt, and what to avoid | ✓ VERIFIED (regression — artifact unchanged) | `docs/research/reference-adapters.md`: 12-row coverage map at v0.89.0, Adopt/Avoid verdicts, Apache-2.0 Attribution, weaker-adapters table, Start-Plan headless → DOCS-03 |

Plan-level must-have truths (28 across the six plans): **28/28 verified** — nothing changed for any truth verdict. The single human-acceptance override (01-02 truth 3, quota framing) carries forward unchanged: ledger line re-read this run (experiments/README.md:189), no-minting half re-grepped (`api_keys` ×0 in probe and scan-secrets; check-docs.sh mint gate lines 145–148 green). The changed tooling truth (01-01 truth 1, secret-shape gate) was re-verified in full this run with the strongest evidence it has had: the gate runs green over the current tree *and* demonstrably still catches every real shape (see Behavioral Spot-Checks).

**Score:** 5/5 ROADMAP SCs; 28/28 plan truths (0 present-but-behavior-unverified)

### Required Artifacts

All 21 artifacts verified at Levels 1–3 (20 declared across the six plans + the supplement's EXP-ORCASRC-01 capture). Mechanical verb result re-run this session: `verify.artifacts` 19/20 declared pass across the six plans (rc=1 only on the one known contains-pattern miss, re-audited below); `experiments/scan-secrets.sh` — the one file that changed — re-verified in full at Levels 1–4.

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `docs/protocol/README.md` | Index + single provenance legend | ✓ VERIFIED | Legend defined exactly once (check (a)); indexes all 5 DISC docs (ids re-grepped this run) |
| `docs/protocol/app-server-dialect.md` | DISC-01 deliverable, 9 sections + Sources | ✓ VERIFIED | All 10 headings exact-match under --phase-gate; drift baseline at line 184 |
| `docs/protocol/acp-client-dialect.md` | DISC-02 deliverable, 10 sections | ✓ VERIFIED | All headings; 6 chunk citation families; repo-citation blocks @v1.4.262; 11 V-RPT preserved; EXP-SESSIONDIR-01 + EXP-ORCASRC-01 both referenced, the latter in gate-covered V-EXP form |
| `docs/protocol/install-discovery.md` | DISC-03 deliverable, 6 sections | ✓ VERIFIED | Matrix + entrypoint + URL-handler clash + auto-update/revision + discovery order + provenance |
| `docs/protocol/workspace-key.md` | DISC-04 deliverable | ✓ VERIFIED | Wire schema / derivation convention / experiment record / host-freedom note / sources |
| `docs/research/reference-adapters.md` | DISC-05 deliverable, 7 sections | ✓ VERIFIED | Coverage map / Adopt / Avoid / `## Attribution` / Weaker adapters / Limitations / Sources |
| `experiments/captures/EXP-ORCASRC-01.jsonl` | Source-cross-verification capture | ✓ VERIFIED | Line-1 header carries `"capture":"EXP-ORCASRC-01"`; gate-covered (check (b)+(f)), headers re-checked this run |
| `experiments/check-docs.sh` | Conformance + phase gate | ✓ VERIFIED | exit 0 both modes re-run at HEAD; checks (a)–(g) pass incl. the api_keys mint gate (lines 145–148); toplevel pin keeps it CWD-independent |
| `experiments/scan-secrets.sh` | Secret-shape gate over tracked files | ✓ VERIFIED (re-verified in full — changed artifact) | exit 0 re-run with the new allowlist; patterns untouched; exact-string allowlist proven necessary (pre-change script fails rc=1 on the Phase-3 fixtures) and proven not to weaken the gate (synthetic battery: JWT / real blob / near-marker variant / token all still fail) |
| `experiments/scrub-capture.mjs` (+ test) | Canonical scrubber + tests | ✓ VERIFIED | 22/22 node --test re-run; escaped-form user_info redaction, boundary-aware home rewrite, header validation all test-covered |
| `experiments/app-server-probe.mjs` | Flow harness | ✓ VERIFIED | flows present; quiet-window end mode; zero-capture exit guard; pre-spawn window validation; request/reply direction guard |
| `experiments/README.md` | Ops guide + quota ledger | ✓ VERIFIED | ledger line states actual spend (3 sends, 0 completed) re-read at line 189; window example 30000 with must-exceed-last-step rule |
| `experiments/captures/EXP-*.jsonl` (6) | Scrubbed captures, headers intact | ✓ VERIFIED | All 6 first lines carry `{"capture":"<id>"}` (check (f) enforces continuously; independently re-checked this run) |
| `.gitignore` | `experiments/raw/` defense | ✓ VERIFIED | Line 37 unchanged |

Mechanical-check notes (re-audited against reality, not waved through):
- `verify.artifacts` on 01-03 reports one miss: plan `contains: "## 3.17 vs 3.18.2 drift baseline"` vs actual heading `## 9. 3.17 vs 3.18.2 drift baseline` (line 184, full drift content — re-grepped this run). The phase's own conformance gate requires and passes exactly the numbered form. Ruled: content requirement satisfied; pattern-literalism false positive, not a stub.
- `verify.key-links` reports 3 misses of pattern `docs/protocol/README` (workspace-key / app-server-dialect / install-discovery). All three reference the legend via sibling-relative links (re-grepped this run: workspace-key.md lines 7/127, app-server-dialect.md line 8, install-discovery.md line 8; reference-adapters via `../protocol/README.md` line 8), and check-docs.sh check (a) enforces "legend defined once, others reference the README" (green this run). Ruled: WIRED; grep-pattern false negative. 01-04's README link verifies mechanically.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| workspace-key.md | EXP-WORKSPACEKEY-01.jsonl | `[V-EXP EXP-WORKSPACEKEY-01]` resolves | WIRED | 13 tag occurrences; capture header intact |
| acp-client-dialect.md | EXP-SESSIONDIR-01 | `[V-EXP EXP-SESSIONDIR-01]` tag | WIRED | citations resolving; capture holds the E5 static answers + `17.0.5` ×3 |
| acp-client-dialect.md | EXP-ORCASRC-01 | `[V-EXP EXP-ORCASRC-01]` tag | WIRED | tag present ×1; gate resolution + header continuity green over it this run |
| acp-client-dialect.md | docs/protocol/README.md | legend referenced | WIRED | mechanically verified |
| app-server-dialect.md | EXP-SUBSCRIBE-01 / EXP-TURN-01 | V-EXP tags | WIRED | 26 + 38 tag occurrences, all resolving |
| install-discovery.md | EXP-DISCOVERY-01 | V-EXP tag | WIRED | 23 occurrences; capture holds dpkg-deb, pristine, computed revision example |
| app-server-probe.mjs | captures | scrub-first pipeline | WIRED | scrub-capture.mjs is the only writer into captures/ (header validation guards it); 22/22 scrubber tests |
| app-server-dialect.md / install-discovery.md / workspace-key.md / reference-adapters.md | docs/protocol/README.md | legend referenced | WIRED | sibling-relative links (mechanical pattern-miss documented above) |
| reference-adapters.md | .planning/research/REFERENCE-ADAPTERS.md | `credential-store` claim verification | WIRED | Golden file named; drift row recorded |
| README.md (protocol) | all five DISC docs | deliverables index | WIRED | Table rows link each doc with its DISC id |

### Data-Flow Trace (Level 4)

Documentation phase: the analog of "rendered value → real data source" is "doc claim → committed evidence".

| Artifact | Claim source | Traced to | Real data | Status |
| -------- | ------------ | --------- | --------- | ------ |
| app-server-dialect.md §4/§5 | V-EXP claims | EXP-SUBSCRIBE-01 / EXP-TURN-01 verdict + wire lines | Live 3.18.2 runs recorded via scrub-first pipeline | ✓ FLOWING |
| workspace-key.md | V-EXP + V-SRC | EXP-WORKSPACEKEY-01 + 3.17.1 clone citations | Live run + version-pinned source | ✓ FLOWING |
| acp-client-dialect.md | V-SRC(chunk) + V-SRC(repo) + V-EXP | 6 named chunk files + tag-pinned v1.4.262 source (EXP-ORCASRC-01 records the reads) + EXP-SESSIONDIR-01 | Fresh extract reads + public-source reads, corrections where source contradicted chunk interpretation | ✓ FLOWING |
| install-discovery.md | V-EXP + V-SRC | EXP-DISCOVERY-01 + bundle literals | Read-only live enumeration + artifact inspection | ✓ FLOWING |
| reference-adapters.md | V-SRC(upstream) | Tag-pinned v0.89.0 clone (scratch, outside repo — by design) | Live study; nothing upstream committed | ✓ FLOWING |
| scan-secrets.sh allowlist | Fixture marker | `internal/zcode/fake/testdata/{golden,scenarios}` (Phase-3 committed fixtures) | Real committed fixture lines carrying the exact self-declaring marker | ✓ FLOWING |
| No static/hardcoded data paths found | — | — | — | none flagged |

### Behavioral Spot-Checks

This verifier re-ran the Phase-1 surface and verified the changed gate in depth. The synthetic battery ran the script's *actual* pattern strings and *actual* allowlist filter over synthetic lines, proving the allowlist drops only the exact marker line.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Docs conformance gate | `bash experiments/check-docs.sh` | exit 0, no CHECK FAIL | ✓ PASS |
| All-six phase gate | `bash experiments/check-docs.sh --phase-gate` | exit 0, 0 PENDING | ✓ PASS |
| Secret-shape gate (changed) | `bash experiments/scan-secrets.sh` | exit 0 over the current tree (marker fixtures present and exempted) | ✓ PASS |
| Allowlist was necessary | pre-change script (`git show f7de0a5:experiments/scan-secrets.sh`) against the current tree | exit 1 — flags `internal/zcode/fake/testdata/golden/credential-cipher-scripted.golden:3` and `scenarios/credential-cipher-scripted.json:17` (the Phase-3 fixtures) under pattern (b) | ✓ PASS (change justified, not a weakening) |
| Allowlist is exactly-string-scoped | synthetic battery through pattern (b) + `grep -vF` filter | marker line dropped; real blob (`enc:v2:os:AbCdEf…`) and near-marker variant (`enc:v2:os:DIFFERENT-not-the-allowlisted-string`) both survive the filter → gate still fails on them | ✓ PASS |
| Gate catching power intact (a)/(c) | synthetic battery through patterns (a)+(c) + filter | JWT shape and `"access_token": "<20+ chars>"` shape survive the filter → gate still fails on them | ✓ PASS |
| Gate prose-honesty | prose line "uses the enc:v2:os: prefix" vs pattern (b) | no match (20+ char body required) | ✓ PASS |
| Scrubber + guard unit suite | `node --test experiments/` | 22 pass / 0 fail / 0 skipped | ✓ PASS |
| MVP goal format | `user-story.validate` on the ROADMAP goal | valid: true, exit 0 | ✓ PASS |
| V-EXP resolution (independent of gate) | bracketed citations vs 6 capture files | all resolve; all 6 headers intact | ✓ PASS |
| No-mint gate | `grep -ciE 'api_keys'` probe + scan-secrets | 0 / 0; check-docs.sh mint gate green in the exit-0 run | ✓ PASS |
| Quota ledger honesty | ledger line (experiments/README.md:189) | "real turns spent: 3 tiny sends issued, 0 completed" | ✓ PASS |
| IN-01 header↔body consistency | header known-unknowns vs Open items unresolved bullets | header names the same three (settings-store path, live --version, interruption); body lists exactly those three, V-RPT-labeled | ✓ PASS |
| IN-02 single claim strength | grep byte-count-identical / byte-identical | build-match claim = byte-count-identical ×3; remaining "byte-identical" hits (lines 103, 528) are different correct claims (AI_VAULT_AGENTS contract; launch registry between source tags) | ✓ PASS |
| Commit validity (changed-file commit) | `verify.commits a49486f` | all_valid: true | ✓ PASS |

The E1–E5 live runs are not re-executable by the verifier without re-spending quota and re-touching live account state; their evidence is the committed captures, content-verified against doc claims. The E-ORCASRC source reads are re-executable by anyone (public repo, tag-pinned citations).

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| docs conformance | `bash experiments/check-docs.sh` | exit 0 | PASS |
| phase gate (strict, all six docs) | `bash experiments/check-docs.sh --phase-gate` | exit 0 | PASS |
| secret scan (changed gate) | `bash experiments/scan-secrets.sh` | exit 0 | PASS |
| unit tests | `node --test experiments/` | 22/22 pass | PASS |

### Requirements Coverage

All five phase requirement IDs are claimed by exactly one plan each and remain satisfied. No orphaned requirements (REQUIREMENTS.md maps only DISC-01..05 to Phase 1; traceability marks all five Complete — re-checked this run at REQUIREMENTS.md:110–116).

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DISC-01 | 01-02 (evidence) + 01-03 (doc) | Event surface documented from source, V-SRC/V-EXP labeled | ✓ SATISFIED | app-server-dialect.md complete; E1 verdicts + E2 honest degradation captured; drift baseline; `session/delete` absence recorded |
| DISC-02 | 01-04 | Client dialect: identity, floor, method sets, full range; spot-check items answered | ✓ SATISFIED | acp-client-dialect.md — both spot-check items answered (session-dir format; modelDiscovery kind `session`, corrected against source @v1.4.262); full method range includes session/close + elicitation/complete; judgment-tier prohibition verified with human acceptance (UAT item 4) and regression-green this run |
| DISC-03 | 01-05 | Desktop discovery matrix incl. auto-update effect on revision | ✓ SATISFIED | install-discovery.md 5-layout matrix; revision formula V-SRC; deb row artifact-inspected |
| DISC-04 | 01-01 | workspaceKey semantics: source + experiment | ✓ SATISFIED | workspace-key.md + EXP-WORKSPACEKEY-01 |
| DISC-05 | 01-06 | Reference implementation compared; findings in docs/research/ | ✓ SATISFIED | reference-adapters.md; live v0.89.0 study; drift recorded; attribution settled |

### Decision Coverage

The `check.decision-coverage-verify` gate RAN this pass (the phase-scoped `01-CONTEXT.md` was passed explicitly): **6/6 trackable decisions honored, not_honored: [], blocking: false** — "All trackable CONTEXT.md decisions are honored by shipped artifacts." D-01 installed-bundle ground truth; D-02 scrub-first pipeline (only writer + 22/22 tests + headers continuously gated + the hardened guards); D-03 docs split per SC; D-04 legend once + inline tags (check (a) green); D-05 targeted-experiments policy incl. honest degradation (ledger line unchanged); D-06 no live chat-UI integration (all this run's executions used local stubs and /tmp scratch — no live card run, no quota).

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| experiments/scrub-capture.test.mjs | D-02 scrub rules (pipeline integrity) | 22 | 0 | 0 | Value/Behavioral (placeholder determinism, byte-identity, whole-value redaction, boundary-aware rewrite, header validation, wrapped-form user_info + fixed point) | SUFFICIENT |

**Disabled tests on requirements:** 0. **Circular patterns:** 0. **Insufficient assertions:** 0. Doc-conformance checks (a)–(g) in check-docs.sh are exercised above, including the two continuous gates from the nyquist audit and the api_keys mint gate. The probe's own guards (WR-01/WR-02/IN-04) were independently reproduced by the prior pass's stub evidence and their committed regression tests remain green in this run's 22/22.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| experiments/scan-secrets.sh (changed) | — | Debt markers TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER | none found | debt-marker gate clean |
| — | — | Debt markers in all other covered implementation files (both dialect docs, remaining experiments/*, .gitignore) | none found (unchanged since prior pass; re-scanned the changed file) | debt-marker gate clean |
| experiments/scrub-capture.mjs, README.md | various | lowercase "placeholder" | ℹ️ Info | scrubber-convention prose/domain language (the scrubber's own redaction placeholders), not stubs |
| — | — | empty implementations / console-log-only / hardcoded empty data | none found | — |

### Advisory (New Scope, Unevidenced)

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | TOKEN_RX (scrub-capture.mjs) matches bare-quote keys only; an escaped-form token value inside the probe envelope is neither scrubbed nor gate-caught (see frontmatter `advisory` for detail) | security | carried forward from the prior report — recorded by the fix batch itself (01-REVIEW-FIX.md follow-up note), batched for the next review pass; re-confirmed unchanged this run; zero current exposure in committed captures |

## Human Verification Required

**None — carried-over items all resolved with machine and/or recorded human evidence:**

1. **MVP goal format** — resolved; `user-story.validate` → valid:true re-confirmed this run; user-approved (UAT item 1).
2. **Provenance spot-check of V-SRC citations** — resolved (UAT item 2, 5/5 claims re-derived).
3. **Quota accounting framing** — accepted by the user (UAT item 3); carried as this report's single override; ledger line re-read this run.
4. **Citation honesty (DISC-02 judgment-tier prohibition)** — resolved via full source cross-verification @v1.4.262 (UAT item 4); consistency discipline re-verified this run (IN-01/IN-02 regression green).
5. **CR-01 / IN-04 fix-report flags** — discharged by machine evidence in the prior pass (planted payload + falsification stub reproduced independently; dedicated regression tests green in this run's 22/22).

No `<verify><human-check>` blocks exist in any plan to harvest (re-scanned this run: 0 across all six plans). No truth is present-but-behavior-unverified: every tooling truth's behavior was exercised by this run (gates executed, suite green, synthetic battery), and the documentation truths' evidence is the committed captures.

## Gaps Summary

None. All 5 ROADMAP success criteria and all 28 plan must-have truths verify against the current tree. The single covered-file change since the prior passed report — the scan-secrets.sh fixture allowlist — was verified in full and makes the phase deliverables *work* in the current tree (Phase-3's committed golden fixtures would otherwise break the CI-reused secret gate) *without weakening it* (proven both directions: pre-change script fails rc=1 on the fixtures; post-change script still fails on every synthetic real shape, including a near-marker variant differing by one substring). The three mechanical-verb misses are the same re-audited pattern-literalism false positives as the prior pass, with content evidence. The carried advisory (TOKEN_RX escaped-form gap) is unchanged, recorded by the fix batch itself with batched follow-up and zero current exposure. Remaining known-unknowns are honestly deferred to Phases 6–7 and covered by later-phase success criteria (AUTH-02 at REQUIREMENTS.md:48/134; SESS-03 at REQUIREMENTS.md:56/146). Status: **passed**.

---

_Verified: 2026-11-20T11:04:00Z_
_Verifier: Claude (gsd-verifier)_
