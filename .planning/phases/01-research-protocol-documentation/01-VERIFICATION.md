---
phase: 01-research-protocol-documentation
verified: 2026-11-19T11:46:21Z
status: passed
score: 5/5 ROADMAP success criteria verified; 28/28 plan must-have truths verified (1 via recorded human-acceptance override)
covered_files: [".gitignore", ".planning/phases/01-research-protocol-documentation/01-01-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-01-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-02-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-02-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-03-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-03-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-04-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-04-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-05-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-05-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-06-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-06-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW-DISPOSITION.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW.md", ".planning/phases/01-research-protocol-documentation/01-SECURITY.md", ".planning/phases/01-research-protocol-documentation/01-UAT.md", ".planning/phases/01-research-protocol-documentation/01-VALIDATION.md", "docs/protocol/README.md", "docs/protocol/acp-client-dialect.md", "docs/protocol/app-server-dialect.md", "docs/protocol/install-discovery.md", "docs/protocol/workspace-key.md", "docs/research/reference-adapters.md", "experiments/README.md", "experiments/app-server-probe.mjs", "experiments/captures/EXP-DISCOVERY-01.jsonl", "experiments/captures/EXP-ORCASRC-01.jsonl", "experiments/captures/EXP-SESSIONDIR-01.jsonl", "experiments/captures/EXP-SUBSCRIBE-01.jsonl", "experiments/captures/EXP-TURN-01.jsonl", "experiments/captures/EXP-WORKSPACEKEY-01.jsonl", "experiments/check-docs.sh", "experiments/scan-secrets.sh", "experiments/scrub-capture.mjs", "experiments/scrub-capture.test.mjs"]
covered_digest: "v4:sha256:b61ef731c1d9ac6b66dcc26c297dd572f48fbd9d298d472a62dc15264a3a6e49"
behavior_unverified: 0
overrides_applied: 1
overrides:
  - must_have: "Exactly one real turn is spent on the account for this phase; the experiments README quota ledger says so; no committed script can mint API keys"
    reason: "The no-minting half is machine-verified (api_keys absent from harness scripts — check-docs.sh check (g), continuous gate). The letter of the spend half did not hold (3 tiny sends issued, 0 completed: first-token latency outran the quiet window twice; mid-turn requestCredentialCipher — keyring material out of Phase-1 scope — blocked completion; 4th run correctly refused). The plan's D-05 degradation branch was executed and ledgered honestly; the human accepted the framing per UAT item 3 (2026-11-19: spend practically zero, 3 truncated requests, no model replies)."
    accepted_by: "user (recorded in 01-UAT.md item 3)"
    accepted_at: "2026-11-19T11:34:57Z"
prohibitions:
  - requirement_id: DISC-02
    statement: "MUST NOT state client-dialect claims as current-build facts without the chunk-filename citation of the extract they were re-verified against; build drift stays visible"
    status: verified
    verification: judgment
    resolution: "Resolved with stronger evidence than the designed human skim: the whole doc was cross-verified against the public client source (github.com/stablyai/orca, MIT) at tag v1.4.262 @13619bc1 — ~20 claim groups dual-cited as [V-SRC <chunk>#<anchor> ... repo src/<path>#<line> @v1.4.262], 3 chunk-only interpretations corrected and flagged in place (commit ed897a8, capture EXP-ORCASRC-01), 3 genuinely-unresolved items stay V-RPT (x11 in doc), build-drift warning in header (line 12) now names both the installed build and the citation tag. Recorded pass in 01-UAT.md item 4."
re_verification:
  previous_status: human_needed
  previous_score: "5/5 ROADMAP success criteria; 27/28 plan truths (1 UNCERTAIN routed to human judgment)"
  gaps_closed:
    - "MVP goal format — ROADMAP Phase 1 goal rewritten to canonical user-story form (commit 860e98d, user-approved; user-story.validate -> valid:true, all slots, re-verified this run)"
    - "Provenance spot-check of V-SRC citations — 5/5 load-bearing claims independently re-derived with machine paths (drift baseline via comm -13 diff; omp argv at byte offsets; workspaceKey at zcode-protocol-entrypoint.ts:227-236; revision digest e9c0674a... reproduced; upstream v0.89.0 no-mint confirmed) — recorded in 01-UAT.md item 2"
    - "Quota accounting framing — accepted by the user as-is (01-UAT.md item 3); carried in this report as the single override"
    - "Citation honesty (DISC-02 judgment-tier prohibition) — resolved via full source cross-verification against public stablyai/orca @v1.4.262 (commits 9b4534b + ed897a8, capture EXP-ORCASRC-01), recorded in 01-UAT.md item 4"
  gaps_remaining: []
  regressions: []
  note: "The stale report's frontmatter status field had been flipped to passed in isolation (commit 4e40c6d) while its body still read human_needed; this report is the actual re-verification that verdict was waiting for. Changed inputs since the previous report: docs/protocol/acp-client-dialect.md + new capture EXP-ORCASRC-01.jsonl (ed897a8), ROADMAP goal line (860e98d), 01-RESEARCH.md accessibility correction (9b4534b), and the phase evidence set (01-SECURITY.md, 01-VALIDATION.md, 01-REVIEW.md, 01-REVIEW-DISPOSITION.md, 01-UAT.md). The fingerprint was recomputed over the current covered set (36 files incl. auto-added PLAN/SUMMARY pairs)."
deferred:
  - truth: "Turn-completion terminal envelopes observed live on the wire (the completion half of app-server-dialect.md section 5 is V-SRC-only by the D-05 degradation)"
    addressed_in: "Phase 6"
    evidence: "app-server-dialect.md section 5: 'source-level only until a credential-answering host exists (Phase 6)'; REQUIREMENTS.md AUTH-02 (Phase 6) covers interaction/requestCredentialCipher"
  - truth: "Known-unknowns: agentCmdOverrides settings-store location, live --version probe half against the real wrapper, interruption semantics beyond the documented strings"
    addressed_in: "Phase 7"
    evidence: "acp-client-dialect.md Open items: all three labeled V-RPT 'Phase 7 live concern' (lines 475-490; EXP-ORCASRC-01 lines 36-38 narrow the first to key+shape confirmed, storage path unknown); REQUIREMENTS.md SESS-03 (Phase 7) covers full dialect coverage live"
advisory:
  - finding: "8 open code-review advisories (0 critical, 4 warning, 4 info) from 01-REVIEW.md, all dispositioned 'open' with explicit batch targets in 01-REVIEW-DISPOSITION.md (Phase 2 CI gate adoption, next doc touch, Phase 3 harness work)"
    category: other
    reason: "Quality advisories on working deliverables with an auditable disposition ledger and scheduled batch targets; none blocks goal achievement"
    evidence_status: "dispositioned — 01-REVIEW-DISPOSITION.md, 8/8 open with batch targets"
---

# Phase 1: Research & Protocol Documentation — Verification Report

**Phase Goal:** As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.
**Verified:** 2026-11-19T11:46:21Z
**Status:** passed
**Re-verification:** Yes — after gap closure / UAT resolution (previous report went stale: covered inputs changed after it was written)

## Re-verification Scope

Changed inputs since the previous report (all post-09:00Z commits): the DISC-02 supplement (docs/protocol/acp-client-dialect.md substantially rewritten + new capture `experiments/captures/EXP-ORCASRC-01.jsonl`, commit ed897a8), the ROADMAP goal line reformatted to canonical user-story form (860e98d), the 01-RESEARCH.md stablyai/orca accessibility correction (9b4534b), and the phase evidence artifacts (01-SECURITY.md, 01-VALIDATION.md, 01-REVIEW.md + disposition, 01-UAT.md). Per re-verification rules: **failed/previously-human items got full 3-level verification; previously-passed items got existence + sanity regression** — all green.

## MVP Mode Guard

`**Mode:** mvp` with the goal now in validated user-story form: `user-story.validate` on the current ROADMAP goal returns **valid: true** (all three slots extracted; re-run this session, exit 0). The previous report's format discrepancy is closed by commit 860e98d + UAT item 1. No refusal path applies.

## User Flow Coverage

User story (ROADMAP goal, verb-validated this run): «As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Open the documentation index | One index names every deliverable doc + the single provenance legend | `docs/protocol/README.md` — deliverables table (5 DISC docs, each naming its DISC id), `## Provenance legend` defined exactly once (grep: only README.md carries the heading; check-docs.sh check (a) green) | ✓ |
| Read the app-server dialect | Full JSONL event surface, subscribe verdict, terminal signals, per-claim provenance | `docs/protocol/app-server-dialect.md` — 10 sections (211 lines, unchanged since prior verify), 110-method map + 43 v4 strings with re-derivable commands/counts, E1 verdict table from `[V-EXP EXP-SUBSCRIBE-01]`, terminal-signal tables (turn-start live; completion honestly V-SRC-only), drift baseline first-class (`## 9.` at line 184) | ✓ |
| Read the client ACP dialect | omp identity, version floor, env quirks, method sets, string contracts — now source-corroborated | `docs/protocol/acp-client-dialect.md` (536 lines, +~325 since prior verify) — omp identity + alternatives ruled out with dual citations; floor 17.0.5 / `resolvesSelectorsFrom:"18.1.9"`; launch contract **corrected against source** (chat argv carries no `--session-dir`; wrapper must accept-and-ignore it for disposable discovery launches); `PI_NO_PTY=1`/`PI_CODING_AGENT_DIR`, full agentCmdOverrides semantics incl. metachar rejection list; startup + per-turn method sets; **19-row** full method range (was 17 — `session/close` and `elicitation/complete` added as NEW); verbatim Hb strings with the -32603/`data.details` matcher precision; `fullAccess` corrected (false only for discovery sessions; chat follows permission mode); modelDiscovery kind `session` corrected (models listed pre-session via disposable probe session); known-unknowns stay V-RPT ×11; build-drift warning names installed build + citation tag v1.4.262 @13619bc1 | ✓ |
| Read the install discovery matrix | Locate commands per layout incl. auto-update effect on the revision string | `docs/protocol/install-discovery.md` — 5-row matrix, deb artifact-inspected `3.18.2-10443`, revision formula V-SRC with computed example `zcode-builtin:44:e9c0674a…`, recommended discovery order (unchanged; regression-green) | ✓ |
| Read the workspaceKey semantics | Source citation + recorded experiment | `docs/protocol/workspace-key.md` — verbatim 3.17.1 derivation quote, nonEmptyString schema lines, `[V-EXP EXP-WORKSPACEKEY-01]` both create verdicts (unchanged; regression-green) | ✓ |
| Read the reference comparison | Coverage, adopt, avoid under docs/research/ | `docs/research/reference-adapters.md` — 12-row coverage map at v0.89.0, golden source+test files, Avoid list, Apache-2.0 attribution, Start-Plan headless note (unchanged; regression-green) | ✓ |
| Outcome: later phases build on verified wire truth | Claims carry resolving provenance; gates enforce it | 106 `[V-EXP …]` citations resolve to the **6** scrubbed captures (headers intact, builtinRevision 44 pinned; the new EXP-ORCASRC-01 header records the registry-bracket build match + v1.4.262 @13619bc1 state); `bash experiments/check-docs.sh --phase-gate` exit 0 with the two nyquist-audit checks (f) header-continuity and (g) no-mint now continuous; later-phase consumption stated per doc; REQUIREMENTS.md DISC-01..05 all marked Complete | ✓ |

## Goal Achievement

### Observable Truths

ROADMAP success criteria (the contract):

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|--------------------|--------|----------|
| 1 | `docs/` protocol documentation describes the app-server JSONL event surface — incl. `session/subscribe`/event-stream availability and turn-completion terminal signals — every claim labeled V-SRC or V-EXP against the shipped 3.18.2 binary | ✓ VERIFIED | `docs/protocol/app-server-dialect.md` §4 subscribe verdict table (session/subscribe ACCEPTED, session/events ACCEPTED, v4/conversation/subscribe REFUSED −32603 — verdict lines present in `EXP-SUBSCRIBE-01.jsonl`); §5 terminal signals (turn-start live `[V-EXP EXP-TURN-01]`; completion half V-SRC-only with the honest D-05 statement). Tag grammar machine-enforced by check-docs.sh check (c); file unchanged since prior verify — regression: heading set exact-match green under --phase-gate |
| 2 | A dialect document names the chosen preset-agent identity with version floor and env quirks, and enumerates the load-bearing method set at card startup/per turn and the full ACP method range | ✓ VERIFIED (re-verified in full — changed artifact) | `docs/protocol/acp-client-dialect.md`: omp identity + ruled-out alternatives, each now dual-cited `[V-SRC <chunk>#<anchor>; repo src/<path>#<line> @v1.4.262]` (18 `repo src/` citation blocks); floor `17.0.5` (doc + EXP-SESSIONDIR-01 capture literal `minimumVersions:[`17.0.5`]` + registry source line #141), `resolvesSelectorsFrom:"18.1.9"`; `PI_NO_PTY=1` with the source-explained reason, `PI_CODING_AGENT_DIR` agent home; startup sequence (initialize → session/new → authenticate error-driven → load/resume conditional) vs per-turn set distinct with overlap note; full range table now **19 rows** incl. the two NEW methods `session/close` (Stop teardown, 10 s timeout) and `elicitation/complete` (URL elicitation finish) — each with repo call-graph citation; Hb string contracts with the matcher precision (code −32603 + string `data.details` only) |
| 3 | A discovery matrix documents how to locate a ZCode desktop install across AppImage / deb / unpacked layouts, including what auto-update does to the builtin revision string | ✓ VERIFIED | `docs/protocol/install-discovery.md` unchanged: 5-row matrix with 23 `[V-EXP EXP-DISCOVERY-01]` citations resolving (capture: 4 dpkg-deb hits, pristine/identical ×2, computed revision example); auto-update section with runtime-tree accretion + revision formula V-SRC; live install/apply step honestly V-RPT |
| 4 | `workspaceKey` derivation semantics documented with both a source citation and a recorded experiment | ✓ VERIFIED | `docs/protocol/workspace-key.md` unchanged: V-SRC citations to 3.17.1 `index.ts:2762/:2784`, `official-mcp-auth.ts:149`, `zcode-protocol-entrypoint.ts:219-236`; `[V-EXP EXP-WORKSPACEKEY-01]` ×13, capture holds both verdicts with header intact |
| 5 | A reference-implementation comparison under `docs/research/` records what the TypeScript zcode-acp project covers (elicitation, slash commands, resume/fork, attachments), what to adopt, and what to avoid | ✓ VERIFIED | `docs/research/reference-adapters.md` unchanged: 12-row coverage map at v0.89.0 incl. elicitation/slash/resume-fork/attachments rows, Adopt (5 golden source + 3 golden test files), Avoid (6 items), Attribution (Apache-2.0 §4(b)/(d)), weaker-adapters table, Start-Plan headless → DOCS-03 |

Plan-level must-have truths (28 across the six plans): **28/28 verified** — the single previously-UNCERTAIN truth (01-02 truth 3, quota framing) is resolved by recorded human acceptance (UAT item 3) and carried as the one override above; its no-minting half remains machine-verified by the now-continuous check-docs.sh check (g). The five changed DISC-02 truths re-verified in full: chunk-filename citations retained for every load-bearing claim (6 chunks named in Sources); card registry re-enumeration intact (8 ACP cards byte-identical to ACP_LAUNCH_SPECS @v1.4.262:155-164; native-zcode verdict "No" with hooks-registry + scan-canon citations; hermes drift note); both open items answered — session-dir format (writer algorithm byte-equal to `PI_CWD_BUCKET_LAYOUT`, "Yes — a Go agent can write omp-compatible history files", with the honest v1-doesn't-advertise-loadSession scope note) and modelDiscovery kind `session` (corrected interpretation flagged in place); known-unknowns stay V-RPT, never presented as verified.

**Score:** 5/5 ROADMAP SCs; 28/28 plan truths (0 present-but-behavior-unverified)

### Required Artifacts

All 21 declared artifacts verified at Levels 1–3 (20 prior + the new capture). Mechanical verb result: `verify.artifacts` 20/21 pass across the six plans; highlights:

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `docs/protocol/README.md` | Index + single provenance legend | ✓ VERIFIED | Legend defined exactly once (check (a)); indexes all 5 DISC docs |
| `docs/protocol/app-server-dialect.md` | DISC-01 deliverable, 9 sections + Sources | ✓ VERIFIED | All 10 headings exact-match under --phase-gate; drift baseline at line 184 |
| `docs/protocol/acp-client-dialect.md` | DISC-02 deliverable, 10 sections | ✓ VERIFIED | All headings; 6 chunk citation families; 18 repo-citation blocks @v1.4.262; 11 V-RPT preserved; EXP-SESSIONDIR-01 + EXP-ORCASRC-01 both referenced |
| `docs/protocol/install-discovery.md` | DISC-03 deliverable, 6 sections | ✓ VERIFIED | Matrix + entrypoint + URL-handler clash + auto-update/revision + discovery order + provenance |
| `docs/protocol/workspace-key.md` | DISC-04 deliverable | ✓ VERIFIED | Wire schema / derivation convention / experiment record / host-freedom note / sources |
| `docs/research/reference-adapters.md` | DISC-05 deliverable, 7 sections | ✓ VERIFIED | Coverage map / Adopt / Avoid / Attribution / Weaker adapters / Limitations / Sources |
| `experiments/captures/EXP-ORCASRC-01.jsonl` | NEW: scrubbed source-cross-verification capture | ✓ VERIFIED | Header line 1 carries `"capture":"EXP-ORCASRC-01"` + build-bracket verdict + v1.4.262 @13619bc1 citation state; 38 lines record 3 corrections (kinds: correction ×3), method-set extension (EXTENDED ×2), 13 CONFIRMED claim groups, 3 remaining UNRESOLVED notes — content matches every doc claim spot-checked |
| `experiments/check-docs.sh` | Conformance + phase gate | ✓ VERIFIED | exit 0 both modes; checks (a)–(g) — (f) capture-header continuity and (g) no-mint are new since prior verify (nyquist audit) and pass |
| `experiments/scan-secrets.sh` | Secret-shape gate | ✓ VERIFIED | exit 0 |
| `experiments/scrub-capture.mjs` (+ test) | Canonical scrubber + tests | ✓ VERIFIED | 12/12 node --test |
| `experiments/app-server-probe.mjs` | Flow harness | ✓ VERIFIED | flows present; quiet-window end mode (unchanged) |
| `experiments/README.md` | Ops guide + quota ledger | ✓ VERIFIED | ledger line states actual spend (3 sends, 0 completed) |
| `experiments/captures/EXP-*.jsonl` (6) | Scrubbed captures, headers intact | ✓ VERIFIED | All 6 first lines carry `{"capture":"<id>",…}` (check (f) enforces continuously); content spot-checks match doc claims |
| `.gitignore` | `experiments/raw/` defense | ✓ VERIFIED | Lines 36–37 |

Mechanical-check notes (re-audited against reality, not waved through):
- `verify.artifacts` on 01-03 reports one miss: plan `contains: "## 3.17 vs 3.18.2 drift baseline"` vs actual heading `## 9. 3.17 vs 3.18.2 drift baseline` (line 184, full drift content). The phase's own conformance gate requires and passes exactly the numbered form. Ruled: content requirement satisfied; pattern-literalism false positive, not a stub.
- `verify.key-links` reports 3 misses of pattern `docs/protocol/README` (workspace-key / app-server-dialect / install-discovery). All three reference the legend via sibling-relative links (`[README.md](README.md)` at lines 7/8/8; reference-adapters via `../protocol/README.md` at line 8), and check-docs.sh check (a) enforces "legend defined once, others reference the README". Ruled: WIRED; grep-pattern false negative. Notably plan 01-04's README link now verifies mechanically — the supplement added explicit `docs/protocol/README.md` mentions (lines 22, 494).

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| workspace-key.md | EXP-WORKSPACEKEY-01.jsonl | `[V-EXP EXP-WORKSPACEKEY-01]` resolves | WIRED | 13 citations; capture header intact |
| acp-client-dialect.md | EXP-SESSIONDIR-01 | `[V-EXP EXP-SESSIONDIR-01]` tag | WIRED | 4 citations resolving; capture holds the E5 static answers + `17.0.5` ×3 + original chunk literals the corrections reconcile against |
| acp-client-dialect.md | EXP-ORCASRC-01 | Sources citation of the source-cross-check capture | WIRED | Named at line 517; capture's correction/verdict records match the doc's flagged corrections 1:1 |
| acp-client-dialect.md | docs/protocol/README.md | legend referenced | WIRED | now also mechanically verified (lines 22, 494) |
| app-server-dialect.md | EXP-SUBSCRIBE-01 / EXP-TURN-01 | V-EXP tags | WIRED | 26 + 38 citations, all resolving |
| install-discovery.md | EXP-DISCOVERY-01 | V-EXP tag | WIRED | 23 citations; capture holds dpkg-deb, pristine, computed revision example |
| app-server-probe.mjs | captures | scrub-first pipeline | WIRED | scrub-capture.mjs is the only writer into captures/; 12/12 scrubber tests |
| app-server-dialect.md / install-discovery.md / workspace-key.md / reference-adapters.md | docs/protocol/README.md | legend referenced | WIRED | sibling-relative links (mechanical pattern-miss documented above) |
| reference-adapters.md | .planning/research/REFERENCE-ADAPTERS.md | `credential-store` claim verification | WIRED | Golden file named ×3; drift row recorded |
| README.md (protocol) | all five DISC docs | deliverables index | WIRED | Table rows link each doc with its DISC id |

### Data-Flow Trace (Level 4)

Documentation phase: the analog of "rendered value → real data source" is "doc claim → committed evidence".

| Artifact | Claim source | Traced to | Real data | Status |
| -------- | ------------ | --------- | --------- | ------ |
| app-server-dialect.md §4/§5 | V-EXP claims | EXP-SUBSCRIBE-01 / EXP-TURN-01 verdict + wire lines | Live 3.18.2 runs recorded via scrub-first pipeline | ✓ FLOWING |
| workspace-key.md | V-EXP + V-SRC | EXP-WORKSPACEKEY-01 + 3.17.1 clone citations | Live run + version-pinned source | ✓ FLOWING |
| acp-client-dialect.md | V-SRC(chunk) + V-SRC(repo) + V-EXP | 6 named chunk files + tag-pinned v1.4.262 source (EXP-ORCASRC-01 records the reads) + EXP-SESSIONDIR-01 | Fresh extract reads + public-source reads, corrections where source contradicted chunk interpretation | ✓ FLOWING |
| install-discovery.md | V-EXP + V-SRC | EXP-DISCOVERY-01 (7 rows) + bundle literals | Read-only live enumeration + artifact inspection | ✓ FLOWING |
| reference-adapters.md | V-SRC(upstream) | Tag-pinned v0.89.0 clone (scratch, outside repo — by design) | Live study; nothing upstream committed (`git ls-files` cross-check clean this run) | ✓ FLOWING |
| No static/hardcoded data paths found | — | — | — | none flagged |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Docs conformance gate | `bash experiments/check-docs.sh` | exit 0, no CHECK FAIL | ✓ PASS |
| All-six phase gate | `bash experiments/check-docs.sh --phase-gate` | exit 0, 0 PENDING | ✓ PASS |
| Secret-shape gate | `bash experiments/scan-secrets.sh` | exit 0 | ✓ PASS |
| Scrubber unit behavior | `node --test experiments/` | 12 pass / 0 fail | ✓ PASS |
| MVP goal format | `user-story.validate` on the ROADMAP goal | valid: true, exit 0 | ✓ PASS |
| NEW-method supplement landed | grep `session/close` / `elicitation/complete` in acp-client-dialect.md | ×2 / ×2, each with repo citations | ✓ PASS |
| Corrections flagged in place | grep "CORRECTED against source" | ×2 blockquotes + fullAccess bullet; EXP-ORCASRC-01 records correction ×3 | ✓ PASS |
| Known-unknowns honesty | grep V-RPT count in acp-client-dialect.md | ×11; 3 open items, never presented as verified | ✓ PASS |
| V-EXP resolution (independent of gate) | 106 bracketed citations vs 6 capture files | all resolve; all 6 headers intact | ✓ PASS |
| Capture-header continuity (check f) | first line of each capture | `"capture":"<id>"` ×6 | ✓ PASS |
| Nothing upstream committed | `git ls-files` cross-check | no upstream/orca/credential-store tracked files | ✓ PASS |
| Commit ancestry (26 hashes incl. post-verify set) | `verify.commits` | invalid: [] | ✓ PASS |

The E1–E5 live runs are not re-executable by the verifier without re-spending quota and re-touching live account state; their evidence is the committed captures, content-verified against doc claims. The E-ORCASRC source reads are re-executable by anyone (public repo, tag-pinned citations) — that re-executability is what resolved the previous citation-honesty human item.

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| docs conformance | `bash experiments/check-docs.sh` | exit 0 | PASS |
| phase gate (strict, all six docs) | `bash experiments/check-docs.sh --phase-gate` | exit 0 | PASS |
| secret scan | `bash experiments/scan-secrets.sh` | exit 0 | PASS |
| unit tests | `node --test experiments/` | 12/12 pass | PASS |

### Requirements Coverage

All five phase requirement IDs are claimed by exactly one plan each and remain satisfied. No orphaned requirements (REQUIREMENTS.md maps only DISC-01..05 to Phase 1; traceability marks all five Complete).

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DISC-01 | 01-02 (evidence) + 01-03 (doc) | Event surface documented from source, V-SRC/V-EXP labeled | ✓ SATISFIED | app-server-dialect.md complete; E1 verdicts + E2 honest degradation captured; drift baseline; `session/delete` absence ×0 recorded |
| DISC-02 | 01-04 | Client dialect: identity, floor, method sets, full range; spot-check items answered | ✓ SATISFIED (strengthened) | acp-client-dialect.md — both remaining items answered (session-dir format; modelDiscovery kind `session`, corrected against source @v1.4.262); full method range now includes session/close + elicitation/complete; REQUIREMENTS.md DISC-02 line's spot-check contract fully discharged |
| DISC-03 | 01-05 | Desktop discovery matrix incl. auto-update effect on revision | ✓ SATISFIED | install-discovery.md 5-layout matrix; revision formula V-SRC; deb row artifact-inspected |
| DISC-04 | 01-01 | workspaceKey semantics: source + experiment | ✓ SATISFIED | workspace-key.md + EXP-WORKSPACEKEY-01 |
| DISC-05 | 01-06 | Reference implementation compared; findings in docs/research/ | ✓ SATISFIED | reference-adapters.md; live v0.89.0 study; drift recorded; attribution settled |

### Decision Coverage

Phase CONTEXT decisions D-01…D-06 re-affirmed honored by shipped artifacts (6/6): D-01 installed-bundle ground truth (app-server-dialect header + captures; the supplement keeps installed-build anchors as ground truth with repo tags as the corroborating half); D-02 scrub-first pipeline (only writer + 12/12 tests + headers intact); D-03 docs split per SC; D-04 legend once + inline tags (check (a) green); D-05 targeted-experiments policy incl. honest degradation; D-06 no live chat-UI integration (the supplement is a static source read of the public repo — no live card run, no quota, per EXP-ORCASRC-01's own note line). The `check.decision-coverage-verify` verb reports `skipped` because it reads a root-level `.planning/CONTEXT.md`, which this project replaces with the phase-scoped 01-CONTEXT.md (unchanged since commit 5398b15, i.e. since before the previous verification) — hence the manual re-affirmation.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| experiments/scrub-capture.test.mjs | D-02 scrub rules (pipeline integrity) | 12 | 0 | 0 | Value/Behavioral (placeholder determinism, byte-identity, whole-value redaction) | SUFFICIENT |

**Disabled tests on requirements:** 0. **Circular patterns:** 0. **Insufficient assertions:** 0. Doc-conformance checks (a)–(g) in check-docs.sh are exercised above, including the two new continuous gates from the nyquist audit.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) in changed files (acp-client-dialect.md, EXP-ORCASRC-01.jsonl, check-docs.sh, 01-RESEARCH.md) | none found | debt-marker gate clean |
| docs/protocol/*.md, experiments/* | various | lowercase "placeholder" | ℹ️ Info | scrubber-convention prose/domain language, not stubs (unchanged from prior scan) |
| — | — | empty implementations / console-log-only / hardcoded empty data | none found | — |

## Human Verification Required

**None — all previous items resolved and recorded in 01-UAT.md (4/4 pass, committed b80be03/ef44442):**

1. **MVP goal format** — resolved: goal rewritten to validated user-story form (860e98d), `user-story.validate` → valid:true re-confirmed this run; user-approved (UAT item 1).
2. **Provenance spot-check of V-SRC citations** — resolved: 5/5 load-bearing claims independently re-derived against the machine-local sources with user-approved agent-run method (UAT item 2).
3. **Quota accounting framing** — resolved: accepted as-is by the user (UAT item 3); carried as this report's single override.
4. **Citation honesty (DISC-02 judgment-tier prohibition)** — resolved with stronger evidence than the designed skim: full source cross-verification against public stablyai/orca @v1.4.262 (ed897a8, EXP-ORCASRC-01), 3 corrections applied, unknowns kept V-RPT (UAT item 4).

No new human items: every truth is verified with machine-checkable or UAT-recorded evidence; nothing is present-but-behavior-unverified; the judgment-tier prohibition has a recorded human-completed check. No `<verify><human-check>` blocks exist in any plan to harvest.

## Gaps Summary

None. All 5 ROADMAP success criteria and all 28 plan must-have truths verify against the current tree; the changed DISC-02 artifact re-verified in full and is *stronger* than at the previous verification (dual chunk+source citations, corrected interpretations, wider method enumeration). The two mechanical-verb misses (1 artifact contains-pattern, 3 link patterns) are re-audited pattern-literalism false positives documented above. Open quality advisories (8) are dispositioned with batch targets. Remaining known-unknowns are honestly deferred to Phases 6–7 and covered by later-phase success criteria (AUTH-02; SESS-03 / Phase 7 SC 3). Status: **passed**.

---

_Verified: 2026-11-19T11:46:21Z_
_Verifier: Claude (gsd-verifier)_
