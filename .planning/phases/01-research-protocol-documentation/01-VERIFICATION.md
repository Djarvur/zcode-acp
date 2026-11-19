---
phase: 01-research-protocol-documentation
verified: 2026-11-19T19:05:00Z
status: passed
score: 5/5 ROADMAP success criteria verified; 28/28 plan must-have truths verified (1 via recorded human-acceptance override)
covered_files: [".gitignore", ".planning/phases/01-research-protocol-documentation/01-01-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-01-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-02-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-02-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-03-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-03-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-04-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-04-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-05-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-05-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-06-PLAN.md", ".planning/phases/01-research-protocol-documentation/01-06-SUMMARY.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW-DISPOSITION.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW-FIX.md", ".planning/phases/01-research-protocol-documentation/01-REVIEW.md", ".planning/phases/01-research-protocol-documentation/01-SECURITY.md", ".planning/phases/01-research-protocol-documentation/01-UAT.md", ".planning/phases/01-research-protocol-documentation/01-VALIDATION.md", "docs/protocol/README.md", "docs/protocol/acp-client-dialect.md", "docs/protocol/app-server-dialect.md", "docs/protocol/install-discovery.md", "docs/protocol/workspace-key.md", "docs/research/reference-adapters.md", "experiments/README.md", "experiments/app-server-probe.mjs", "experiments/captures/EXP-DISCOVERY-01.jsonl", "experiments/captures/EXP-ORCASRC-01.jsonl", "experiments/captures/EXP-SESSIONDIR-01.jsonl", "experiments/captures/EXP-SUBSCRIBE-01.jsonl", "experiments/captures/EXP-TURN-01.jsonl", "experiments/captures/EXP-WORKSPACEKEY-01.jsonl", "experiments/check-docs.sh", "experiments/scan-secrets.sh", "experiments/scrub-capture.mjs", "experiments/scrub-capture.test.mjs"]
covered_digest: "v4:sha256:52029d577ee375863bca7d36da1adac4cc00b33253ff16e34408dcbb73990cff"
behavior_unverified: 0
overrides_applied: 1
overrides:
  - must_have: "Exactly one real turn is spent on the account for this phase; the experiments README quota ledger says so; no committed script can mint API keys"
    reason: "The no-minting half is machine-verified (api_keys absent from harness scripts — re-grepped this run: 0 hits in probe + scan-secrets; check-docs.sh check (g), continuous gate). The letter of the spend half did not hold (3 tiny sends issued, 0 completed: first-token latency outran the quiet window twice; mid-turn requestCredentialCipher — keyring material out of Phase-1 scope — blocked completion; 4th run correctly refused). The plan's D-05 degradation branch was executed and ledgered honestly (ledger line re-read this run: 'real turns spent: 3 tiny sends issued, 0 completed'); the human accepted the framing per UAT item 3 (2026-11-19: spend practically zero, 3 truncated requests, no model replies)."
    accepted_by: "user (recorded in 01-UAT.md item 3)"
    accepted_at: "2026-11-19T11:34:57Z"
prohibitions:
  - requirement_id: DISC-02
    statement: "MUST NOT state client-dialect claims as current-build facts without the chunk-filename citation of the extract they were re-verified against; build drift stays visible"
    status: verified
    verification: judgment
    resolution: "Resolved with stronger evidence than the designed human skim: the whole doc was cross-verified against the public client source (github.com/stablyai/orca, MIT) at tag v1.4.262 @13619bc1 — ~20 claim groups dual-cited as [V-SRC <chunk>#<anchor> ... repo src/<path>#<line> @v1.4.262], 3 chunk-only interpretations corrected and flagged in place (commit ed897a8, capture EXP-ORCASRC-01), 3 genuinely-unresolved items stay V-RPT (x11 in doc, re-counted this run), build-drift warning in header names both the installed build and the citation tag. Since the last report, the re-review/fix batch (f199946/9e9406d/1efe48c) tightened exactly this discipline further: header known-unknowns count aligned with the Open items list (three), the build-match claim stated at one strength (byte-count-identical, with the Sources section explaining the exact claim strength), and EXP-ORCASRC-01 promoted to [V-EXP] tag form so the resolution + capture-header continuity gates cover it. Recorded pass in 01-UAT.md item 4."
re_verification:
  previous_status: passed
  previous_score: "5/5 ROADMAP success criteria; 28/28 plan truths (1 override)"
  gaps_closed: []
  gaps_remaining: []
  regressions: []
  note: "Staleness refresh (#4682), not a gap-closure round: the prior verdict was passed and stays passed. Changed covered inputs since the prior report, in two rounds. (A) Phase-2 review-fix work on Phase-1 covered files (commits 91bee67, 52f660a/b886a32, 4ed831a/3969eae, 8959c9d, 527e683, 16:40-18:03 local): .gitignore +4 (go build output — Phase-2 lint gate), scrubber boundary-aware home rewrite + capture-header validation (WR-03/IN-04 round 1, +3 tests) + CR-01 lastIndex fix (+2 tests), scan-secrets.sh pattern alignment with the scrubber + toplevel pin, check-docs.sh toplevel pin, app-server-probe.mjs --desktop-version flag + time-honest header, app-server-dialect.md re-derivable recipes (digit-class fix, singleton count 8->9, grep -e note). (B) The Phase-1 deep re-review (01-REVIEW.md, 17:59Z, 7 findings: 1C/2W/4I) and its fix batch (641502f..1be656b, merge 8a1933d): CR-01 wrapped-form user_info redaction, WR-01 zero-capture exit guard, WR-02 window example + pre-spawn validation, IN-01..04 doc/probe precision fixes; 01-REVIEW-DISPOSITION.md now 9/9 fixed, 0 open; 01-REVIEW-FIX.md records the batch; 01-VALIDATION.md re-audit at HEAD (f3c35d1, suite green, 0 gaps). All changed files re-verified in full (Levels 1-4 + behavioral); unchanged files regression-checked. Fingerprint recomputed over the current 37-file covered set (incl. the new 01-REVIEW-FIX.md). The failing `go test ./...` in internal/zcode is the parallel session's in-flight TDD RED phase for plan 03-02 (3eef389/a4c631b/f67e21d/6c3bcbe, dirty call_test.go) — Phase 1 has no Go code and no Go-based must-haves; this verification ran and judged the Phase-1 surface (check-docs.sh, scan-secrets.sh, node --test experiments/) itself."
deferred:
  - truth: "Turn-completion terminal envelopes observed live on the wire (the completion half of app-server-dialect.md section 5 is V-SRC-only by the D-05 degradation)"
    addressed_in: "Phase 6"
    evidence: "app-server-dialect.md section 5: 'source-level only until a credential-answering host exists (Phase 6)'; REQUIREMENTS.md AUTH-02 (Phase 6, Pending) covers interaction/requestCredentialCipher (re-checked this run at REQUIREMENTS.md:48,134)"
  - truth: "Known-unknowns: agentCmdOverrides settings-store location, live --version probe half against the real wrapper, interruption semantics beyond the documented strings"
    addressed_in: "Phase 7"
    evidence: "acp-client-dialect.md Open items: all three labeled V-RPT 'Phase 7 live concern' (EXP-ORCASRC-01 narrowed the first to key+shape confirmed, storage path unknown); REQUIREMENTS.md SESS-03 (Phase 7) covers full dialect coverage live"
advisory:
  - finding: "TOKEN_RX in experiments/scrub-capture.mjs (line 54) matches bare-quote keys only — an escaped-form \\\"access_token\\\":\\\"<20+ chars>\\\" inside the probe envelope is neither scrubbed nor caught by scan-secrets.sh pattern (c) (whose literal closing-quote requirement also fails on the escaped backslash-quote sequence). JWT/credential/UUID rules are shape-based and safe."
    category: security
    reason: "Recorded by the fix batch itself as a follow-up lead in 01-REVIEW-FIX.md ('Notes for follow-up'), batched for the next review pass; no test currently exercises the escaped-form token shape, so it is flagged rather than counted against any must-have. Exposure is narrow: bare 20+ char token VALUES in escaped envelope form; the committed captures contain zero user_info/token payloads (swept by 01-REVIEW.md)."
    evidence_status: "recorded — 01-REVIEW-FIX.md follow-up note; regex inspected this run (TOKEN_RX line 54; scan-secrets pattern (c) closing-quote requirement)"
---

# Phase 1: Research & Protocol Documentation — Verification Report

**Phase Goal:** As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.
**Verified:** 2026-11-19T19:05:00Z
**Status:** passed
**Re-verification:** Yes — staleness refresh (#4682): covered files changed after the prior (passed) report; verdict re-derived against the current tree

## Re-verification Scope

Changed inputs since the previous report (both post-11:47Z): **(A)** the Phase-2 review-fix batch on Phase-1 covered files (`.gitignore` +4, scrubber boundary/header work +5 tests, gate pattern alignment + toplevel pins, probe `--desktop-version`, app-server-dialect.md recipe corrections — commits 91bee67…527e683) and **(B)** the Phase-1 deep re-review (01-REVIEW.md, 7 findings: 1C/2W/4I) plus its fix batch integrated by merge 8a1933d (CR-01, WR-01/02, IN-01..04 — touching `docs/protocol/acp-client-dialect.md`, `experiments/{app-server-probe.mjs, check-docs.sh, scan-secrets.sh, scrub-capture.mjs, scrub-capture.test.mjs}`, `experiments/README.md`), with the disposition ledger now 9/9 fixed / 0 open, the new 01-REVIEW-FIX.md, and the 01-VALIDATION.md HEAD re-audit. Per re-verification rules: **changed items got full Levels 1–4 + behavioral verification; unchanged items got existence + sanity regression** — all green. The verdict is re-derived from this run's own evidence, not inherited.

## MVP Mode Guard

`**Mode:** mvp` with the goal in validated user-story form: `user-story.validate` on the current ROADMAP goal returns **valid: true** (re-run this session, exit 0). No refusal path applies.

## User Flow Coverage

User story (ROADMAP goal, verb-validated this run): «As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Open the documentation index | One index names every deliverable doc + the single provenance legend | `docs/protocol/README.md` — deliverables table (5 DISC docs, each naming its DISC id), `## Provenance legend` defined exactly once (check-docs.sh check (a) green this run) | ✓ |
| Read the app-server dialect | Full JSONL event surface, subscribe verdict, terminal signals, per-claim provenance | `docs/protocol/app-server-dialect.md` — 10 sections, 110-method map + 43 v4 strings with **corrected, re-derivable recipes** (extraction class now `[a-zA-Z0-9/]*` so the digit-bearing v4 strings survive; singleton row now says (9) and lists 9; error-code counts documented with the load-bearing `grep -e` form), E1 verdict table from `[V-EXP EXP-SUBSCRIBE-01]` (22 citations, all resolving), terminal-signal tables (turn-start live; completion honestly V-SRC-only with the D-05 statement re-read this run at lines 111/128), drift baseline first-class (`## 9.` at line 184) | ✓ |
| Read the client ACP dialect | omp identity, version floor, env quirks, method sets, string contracts — source-corroborated | `docs/protocol/acp-client-dialect.md` — omp identity + alternatives ruled out with dual citations; floor 17.0.5 (doc ×2 + EXP-SESSIONDIR-01 capture ×3); launch contract corrected against source; `PI_NO_PTY=1`/`PI_CODING_AGENT_DIR`, agentCmdOverrides semantics; startup + per-turn method sets; 19-row full method range incl. `session/close` + `elicitation/complete` (4 lines, repo call-graph citations); verbatim Hb strings with the -32603/`data.details` matcher precision; **header now consistent with the body** (IN-01: "three remaining known-unknowns" ×1, Open items lists exactly 3; IN-02: build-match claim at one strength — byte-count-identical at lines 17/524/527, the two remaining "byte-identical" hits are different correct claims about file content); **EXP-ORCASRC-01 now gate-covered** (IN-03: `[V-EXP EXP-ORCASRC-01]` tag ×1, header intact); known-unknowns stay V-RPT ×11 | ✓ |
| Read the install discovery matrix | Locate commands per layout incl. auto-update effect on the revision string | `docs/protocol/install-discovery.md` unchanged: 5-row matrix, 16 `[V-EXP EXP-DISCOVERY-01]` citations resolving (capture header intact), revision formula V-SRC, recommended discovery order | ✓ |
| Read the workspaceKey semantics | Source citation + recorded experiment | `docs/protocol/workspace-key.md` unchanged: verbatim 3.17.1 derivation quotes, 7 `[V-EXP EXP-WORKSPACEKEY-01]` citations resolving, capture header intact | ✓ |
| Read the reference comparison | Coverage, adopt, avoid under docs/research/ | `docs/research/reference-adapters.md` unchanged: 12-row coverage map at v0.89.0, golden source+test files, Avoid list, `## Attribution` present | ✓ |
| Outcome: later phases build on verified wire truth | Claims carry resolving provenance; gates enforce it | All `[V-EXP …]` citations resolve to the 6 scrubbed captures (all 6 headers carry `{"capture":"<id>"}`, check (f) continuous and now covering EXP-ORCASRC-01); `bash experiments/check-docs.sh --phase-gate` exit 0 **re-run by this verifier at HEAD**; `scan-secrets.sh` exit 0 re-run; `node --test experiments/` 22/22 re-run; later-phase consumption stated per doc; REQUIREMENTS.md DISC-01..05 all marked Complete | ✓ |

## Goal Achievement

### Observable Truths

ROADMAP success criteria (the contract):

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|--------------------|--------|----------|
| 1 | `docs/` protocol documentation describes the app-server JSONL event surface — incl. `session/subscribe`/event-stream availability and turn-completion terminal signals — every claim labeled V-SRC or V-EXP against the shipped 3.18.2 binary | ✓ VERIFIED (re-verified in full — changed artifact) | `docs/protocol/app-server-dialect.md` §4 subscribe verdict table (session/subscribe ACCEPTED, session/events ACCEPTED, v4/conversation/subscribe REFUSED −32603; EXP-SUBSCRIBE-01 citations ×22 all resolving); §5 terminal signals (turn-start live `[V-EXP EXP-TURN-01]` ×30; completion honestly V-SRC-only with the D-05 honesty statement). Tag grammar machine-enforced by check-docs.sh check (c); this run's changes (digit-class extraction recipe, singleton count 9/9 consistent — counted, grep `-e` note with the GNU-grep rc=1 pitfall documented) make the doc's recipes *more* re-derivable, and each was verified this run (synthetic-shape recipe greps; count-vs-list match) |
| 2 | A dialect document names the chosen preset-agent identity with version floor and env quirks, and enumerates the load-bearing method set at card startup/per turn and the full ACP method range | ✓ VERIFIED (re-verified in full — changed artifact) | `docs/protocol/acp-client-dialect.md`: omp identity + ruled-out alternatives, dual-cited chunk + repo @v1.4.262; floor `17.0.5`; `PI_NO_PTY=1` source-explained, `PI_CODING_AGENT_DIR`; startup vs per-turn sets distinct; full range table 19 rows incl. `session/close` + `elicitation/complete` with repo citations; Hb string contracts with matcher precision; the re-review's three doc findings fixed and verified this run (header count = Open items count = 3; single claim strength byte-count-identical with the Sources explanation; EXP-ORCASRC-01 in V-EXP tag form — resolution + header-continuity gates green over it) |
| 3 | A discovery matrix documents how to locate a ZCode desktop install across AppImage / deb / unpacked layouts, including what auto-update does to the builtin revision string | ✓ VERIFIED | `docs/protocol/install-discovery.md` unchanged (regression-green): 5-row matrix with 16 resolving `[V-EXP EXP-DISCOVERY-01]` citations; auto-update section with runtime-tree accretion + revision formula V-SRC; live install/apply step honestly V-RPT |
| 4 | `workspaceKey` derivation semantics documented with both a source citation and a recorded experiment | ✓ VERIFIED | `docs/protocol/workspace-key.md` unchanged (regression-green): V-SRC citations to 3.17.1 sources; `[V-EXP EXP-WORKSPACEKEY-01]` ×7 resolving, capture header intact |
| 5 | A reference-implementation comparison under `docs/research/` records what the TypeScript zcode-acp project covers (elicitation, slash commands, resume/fork, attachments), what to adopt, and what to avoid | ✓ VERIFIED | `docs/research/reference-adapters.md` unchanged (regression-green): 12-row coverage map at v0.89.0, Adopt/Avoid verdicts, Apache-2.0 Attribution, weaker-adapters table, Start-Plan headless → DOCS-03 |

Plan-level must-have truths (28 across the six plans): **28/28 verified** — nothing changed for any truth verdict; the single human-acceptance override (01-02 truth 3, quota framing) carries forward unchanged (UAT item 3; ledger line re-read this run; no-minting half re-grepped: `api_keys` ×0 in probe and scan-secrets, check (g) green). The two heavily-changed tooling truths re-verified behaviorally this run (see Behavioral Spot-Checks: CR-01/WR-01/WR-02/IN-04 all reproduced independently by this verifier with the review's own payloads/stubs, plus the committed regression tests green in the 22/22 suite).

**Score:** 5/5 ROADMAP SCs; 28/28 plan truths (0 present-but-behavior-unverified)

### Required Artifacts

All 21 artifacts verified at Levels 1–3 (20 declared across the six plans + the supplement's EXP-ORCASRC-01 capture). Mechanical verb result re-run this session: `verify.artifacts` 20/20 declared pass across the six plans, with the one known contains-pattern miss:

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `docs/protocol/README.md` | Index + single provenance legend | ✓ VERIFIED | Legend defined exactly once (check (a)); indexes all 5 DISC docs |
| `docs/protocol/app-server-dialect.md` | DISC-01 deliverable, 9 sections + Sources | ✓ VERIFIED | All 10 headings exact-match under --phase-gate; drift baseline at line 184 |
| `docs/protocol/acp-client-dialect.md` | DISC-02 deliverable, 10 sections | ✓ VERIFIED | All headings; 6 chunk citation families; repo-citation blocks @v1.4.262; 11 V-RPT preserved; EXP-SESSIONDIR-01 + EXP-ORCASRC-01 both referenced — the latter now in gate-covered V-EXP form |
| `docs/protocol/install-discovery.md` | DISC-03 deliverable, 6 sections | ✓ VERIFIED | Matrix + entrypoint + URL-handler clash + auto-update/revision + discovery order + provenance |
| `docs/protocol/workspace-key.md` | DISC-04 deliverable | ✓ VERIFIED | Wire schema / derivation convention / experiment record / host-freedom note / sources |
| `docs/research/reference-adapters.md` | DISC-05 deliverable, 7 sections | ✓ VERIFIED | Coverage map / Adopt / Avoid / `## Attribution` / Weaker adapters / Limitations / Sources |
| `experiments/captures/EXP-ORCASRC-01.jsonl` | Source-cross-verification capture | ✓ VERIFIED | Line-1 header carries `"capture":"EXP-ORCASRC-01"` + build-bracket verdict + v1.4.262 @13619bc1 state; now continuously gate-covered (check (b)+(f)) |
| `experiments/check-docs.sh` | Conformance + phase gate | ✓ VERIFIED | exit 0 both modes re-run at HEAD; checks (a)–(g) pass; new toplevel pin makes it CWD-independent |
| `experiments/scan-secrets.sh` | Secret-shape gate | ✓ VERIFIED | exit 0 re-run; patterns re-aligned with the scrubber (separator-inclusive CRED class; quote-free token match) — synthetic-shape sanity re-checked this run (JWT/dotted blob/legacy v1/line-final token match; prose prefix and short values don't) |
| `experiments/scrub-capture.mjs` (+ test) | Canonical scrubber + tests | ✓ VERIFIED | 22/22 node --test re-run; escaped-form user_info redaction, boundary-aware home rewrite, header validation all test-covered |
| `experiments/app-server-probe.mjs` | Flow harness | ✓ VERIFIED | flows present; quiet-window end mode; zero-capture exit guard; pre-spawn window validation; request/reply direction guard |
| `experiments/README.md` | Ops guide + quota ledger | ✓ VERIFIED | ledger line states actual spend (3 sends, 0 completed); window example now 30000 with the must-exceed-last-step rule documented |
| `experiments/captures/EXP-*.jsonl` (6) | Scrubbed captures, headers intact | ✓ VERIFIED | All 6 first lines carry `{"capture":"<id>"}` (check (f) enforces continuously); content spot-checks match doc claims |
| `.gitignore` | `experiments/raw/` defense | ✓ VERIFIED | Line 36–37 unchanged; +4 lines are Phase-2's go-build-output entry (benign to Phase-1 truths) |

Mechanical-check notes (re-audited against reality, not waved through):
- `verify.artifacts` on 01-03 reports one miss: plan `contains: "## 3.17 vs 3.18.2 drift baseline"` vs actual heading `## 9. 3.17 vs 3.18.2 drift baseline` (line 184, full drift content — re-grepped this run). The phase's own conformance gate requires and passes exactly the numbered form. Ruled: content requirement satisfied; pattern-literalism false positive, not a stub.
- `verify.key-links` reports 3 misses of pattern `docs/protocol/README` (workspace-key / app-server-dialect / install-discovery). All three reference the legend via sibling-relative links (re-grepped this run: lines 7/7/7 + acp line 23; reference-adapters via `../protocol/README.md`), and check-docs.sh check (a) enforces "legend defined once, others reference the README". Ruled: WIRED; grep-pattern false negative. 01-04's README link verifies mechanically (lines 22, 494).

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| workspace-key.md | EXP-WORKSPACEKEY-01.jsonl | `[V-EXP EXP-WORKSPACEKEY-01]` resolves | WIRED | 7 citations; capture header intact |
| acp-client-dialect.md | EXP-SESSIONDIR-01 | `[V-EXP EXP-SESSIONDIR-01]` tag | WIRED | citations resolving; capture holds the E5 static answers + `17.0.5` ×3 |
| acp-client-dialect.md | EXP-ORCASRC-01 | `[V-EXP EXP-ORCASRC-01]` tag (IN-03 fix) | WIRED | tag present ×1; gate resolution + header continuity green over it this run |
| acp-client-dialect.md | docs/protocol/README.md | legend referenced | WIRED | mechanically verified (lines 22, 494) |
| app-server-dialect.md | EXP-SUBSCRIBE-01 / EXP-TURN-01 | V-EXP tags | WIRED | 22 + 30 citations, all resolving |
| install-discovery.md | EXP-DISCOVERY-01 | V-EXP tag | WIRED | 16 citations; capture holds dpkg-deb, pristine, computed revision example |
| app-server-probe.mjs | captures | scrub-first pipeline | WIRED | scrub-capture.mjs is the only writer into captures/ (header validation now guards it); 22/22 scrubber tests |
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
| reference-adapters.md | V-SRC(upstream) | Tag-pinned v0.89.0 clone (scratch, outside repo — by design) | Live study; nothing upstream committed (`git ls-files` cross-check clean this run) | ✓ FLOWING |
| No static/hardcoded data paths found | — | — | — | none flagged |

### Behavioral Spot-Checks

This verifier re-ran the Phase-1 surface and independently reproduced the review-fix batch's key behaviors (the fix report's "requires human verification" flags on CR-01 and IN-04 are discharged here by machine evidence: this verifier ran the review's exact planted payload and exact falsification stub, and the committed suite now carries dedicated regression tests for both).

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Docs conformance gate | `bash experiments/check-docs.sh` | exit 0, no CHECK FAIL | ✓ PASS |
| All-six phase gate | `bash experiments/check-docs.sh --phase-gate` | exit 0, 0 PENDING | ✓ PASS |
| Secret-shape gate | `bash experiments/scan-secrets.sh` | exit 0 | ✓ PASS |
| Scrubber + guard unit suite | `node --test experiments/` | 22 pass / 0 fail | ✓ PASS |
| MVP goal format | `user-story.validate` on the ROADMAP goal | valid: true, exit 0 | ✓ PASS |
| CR-01: wrapped-form user_info redaction | scrubber on the review's planted `{"ts",…,"line":"{…\"user_info\":{\"email\":\"leak@example.test\",\"marker\":\"WRAPPED_MARKER_XYZ\"}…}"}` capture | `[REDACTED-USERINFO]` present; zero residue for email/marker; envelope parses; re-scrub is a fixed point (byte-identical) | ✓ PASS |
| WR-01: zero-capture run fails | probe with crash-exit stub (exit 1, zero lines), flow=probe | exit 1 with the stdin-EOF-pitfall warning (pre-fix behavior was exit 0) | ✓ PASS |
| WR-02: window shorter than last step refused | probe `--flow subscribe --window-ms 12000` | exit 1 naming the 13000 ms last step; no raw file written (pre-spawn guard) | ✓ PASS |
| IN-04: request frame never counts as a reply | probe flow=subscribe, stub sends `{"id":3,"method":"interaction/requestPermission"}` (collides with session/subscribe probe id) | verdict line: `"probeId":3,"verdict":"no-reply"` for all three probed methods — honest no-reply, no falsified "accepted" | ✓ PASS |
| Secret-gate pattern sanity | synthetic shapes vs the three gate patterns | JWT / dotted `enc:v2:os:` blob / legacy `enc:v1:` / line-final token all match; prose prefix mention and short values don't | ✓ PASS |
| IN-01: header/Open-items count consistency | grep "three remaining" + Open items bullets | header ×1; Open items list exactly 3 known-unknowns | ✓ PASS |
| IN-02: one claim strength | grep byte-identical / byte-count-identical | build-match claim = byte-count-identical (lines 17/524/527); remaining "byte-identical" hits are different correct claims | ✓ PASS |
| Singleton count consistency | count methods in the (9) row | 9 listed = label (9) | ✓ PASS |
| V-EXP resolution (independent of gate) | bracketed citations vs 6 capture files | all resolve; all 6 headers intact | ✓ PASS |
| No-mint gate | `grep -ciE 'api_keys'` probe + scan-secrets | 0 / 0 | ✓ PASS |
| Quota ledger honesty | ledger line | "real turns spent: 3 tiny sends issued, 0 completed" | ✓ PASS |
| Nothing upstream committed | `git ls-files` cross-check | no upstream/orca/credential-store tracked files | ✓ PASS |
| Commit ancestry (25 hashes incl. both fix rounds + parallel 03 commits) | `verify.commits` | invalid: [] | ✓ PASS |

The E1–E5 live runs are not re-executable by the verifier without re-spending quota and re-touching live account state; their evidence is the committed captures, content-verified against doc claims. The E-ORCASRC source reads are re-executable by anyone (public repo, tag-pinned citations).

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| docs conformance | `bash experiments/check-docs.sh` | exit 0 | PASS |
| phase gate (strict, all six docs) | `bash experiments/check-docs.sh --phase-gate` | exit 0 | PASS |
| secret scan | `bash experiments/scan-secrets.sh` | exit 0 | PASS |
| unit tests | `node --test experiments/` | 22/22 pass | PASS |

### Requirements Coverage

All five phase requirement IDs are claimed by exactly one plan each and remain satisfied. No orphaned requirements (REQUIREMENTS.md maps only DISC-01..05 to Phase 1; traceability marks all five Complete).

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| DISC-01 | 01-02 (evidence) + 01-03 (doc) | Event surface documented from source, V-SRC/V-EXP labeled | ✓ SATISFIED | app-server-dialect.md complete; E1 verdicts + E2 honest degradation captured; drift baseline; `session/delete` absence recorded; this run's recipe corrections strengthen re-derivability |
| DISC-02 | 01-04 | Client dialect: identity, floor, method sets, full range; spot-check items answered | ✓ SATISFIED (strengthened) | acp-client-dialect.md — both spot-check items answered (session-dir format; modelDiscovery kind `session`, corrected against source @v1.4.262); full method range includes session/close + elicitation/complete; re-review doc fixes landed (count/strength/citation-gate consistency) |
| DISC-03 | 01-05 | Desktop discovery matrix incl. auto-update effect on revision | ✓ SATISFIED | install-discovery.md 5-layout matrix; revision formula V-SRC; deb row artifact-inspected |
| DISC-04 | 01-01 | workspaceKey semantics: source + experiment | ✓ SATISFIED | workspace-key.md + EXP-WORKSPACEKEY-01 |
| DISC-05 | 01-06 | Reference implementation compared; findings in docs/research/ | ✓ SATISFIED | reference-adapters.md; live v0.89.0 study; drift recorded; attribution settled |

### Decision Coverage

Phase CONTEXT decisions D-01…D-06 re-affirmed honored by shipped artifacts (6/6): D-01 installed-bundle ground truth (app-server-dialect header + captures; the supplement keeps installed-build anchors as ground truth with repo tags as the corroborating half — IN-02's single-claim-strength fix makes the corroboration hierarchy exact); D-02 scrub-first pipeline (only writer + 22/22 tests + headers continuously gated + the CR-01/WR-03/IN-04 hardening); D-03 docs split per SC; D-04 legend once + inline tags (check (a) green); D-05 targeted-experiments policy incl. honest degradation (ledger line unchanged); D-06 no live chat-UI integration (all this run's probe executions used local stubs and /tmp scratch — no live card run, no quota). The `check.decision-coverage-verify` verb reports `skipped` because it reads a root-level `.planning/CONTEXT.md`, which this project replaces with the phase-scoped 01-CONTEXT.md (unchanged since commit 5398b15, re-checked this run) — hence the manual re-affirmation.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| experiments/scrub-capture.test.mjs | D-02 scrub rules (pipeline integrity) | 22 | 0 | 0 | Value/Behavioral (placeholder determinism, byte-identity, whole-value redaction, boundary-aware rewrite, header validation, wrapped-form user_info + fixed point) | SUFFICIENT |

**Disabled tests on requirements:** 0. **Circular patterns:** 0. **Insufficient assertions:** 0. Doc-conformance checks (a)–(g) in check-docs.sh are exercised above, including the two continuous gates from the nyquist audit. The probe's own guards (WR-01/WR-02/IN-04) are additionally covered by this verifier's independent stub reproductions recorded in Behavioral Spot-Checks.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) in all changed files (both dialect docs, experiments/*, .gitignore) | none found | debt-marker gate clean |
| experiments/scrub-capture.mjs, README.md | various | lowercase "placeholder" | ℹ️ Info | scrubber-convention prose/domain language (the scrubber's own redaction placeholders), not stubs |
| — | — | empty implementations / console-log-only / hardcoded empty data | none found | — |

### Advisory (New Scope, Unevidenced)

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | TOKEN_RX (scrub-capture.mjs:54) matches bare-quote keys only; an escaped-form token value inside the probe envelope is neither scrubbed nor gate-caught (see frontmatter `advisory` for detail) | security | new-scope lead recorded by the fix batch itself (01-REVIEW-FIX.md follow-up note), batched for the next review pass; zero current exposure in committed captures |

Prior-report advisories resolved: the 8 open code-review advisories are gone — the re-review verified all prior findings fixed empirically, and the new 7 findings are all fixed with 01-REVIEW-DISPOSITION.md now reading 9/9 fixed, 0 open.

## Human Verification Required

**None — carried over and new flags all resolved with machine evidence:**

1. **MVP goal format** — resolved (860e98d); `user-story.validate` → valid:true re-confirmed this run; user-approved (UAT item 1).
2. **Provenance spot-check of V-SRC citations** — resolved (UAT item 2, 5/5 claims re-derived).
3. **Quota accounting framing** — accepted by the user (UAT item 3); carried as this report's single override; ledger line re-read this run.
4. **Citation honesty (DISC-02 judgment-tier prohibition)** — resolved via full source cross-verification @v1.4.262 (UAT item 4); further tightened this cycle by IN-01/02/03 (count/strength/gate-coverage consistency).
5. **CR-01 "requires human verification" flag (fix report)** — discharged by machine evidence: this verifier ran the review's exact planted wrapped payload (redacted wholesale, zero residue, fixed point, parseable envelope) and the committed suite carries the wrapped-form regression + fixed-point tests (green in 22/22).
6. **IN-04 "requires human verification" flag (fix report)** — discharged by machine evidence: this verifier reproduced the review's exact falsification stub (method-bearing frame colliding with probe id 3 on the subscribe flow); the verdict is honest `no-reply`, not the pre-fix falsified `accepted`.

No `<verify><human-check>` blocks exist in any plan to harvest. No truth is present-but-behavior-unverified.

## Gaps Summary

None. All 5 ROADMAP success criteria and all 28 plan must-have truths verify against the current tree. The two post-verification change rounds made the phase deliverables *stronger*, not weaker: the scrub-first pipeline is hardened (escaped-form redaction, boundary-aware rewrite, header validation, zero-capture/window guards, request/reply direction guard — all behaviorally verified this run), and both dialect docs' precision discipline was tightened (re-derivable recipes, count/strength consistency, gate-covered capture citations). The three mechanical-verb misses (1 artifact contains-pattern, 3 link patterns) are re-audited pattern-literalism false positives with content evidence. The one new advisory (TOKEN_RX escaped-form gap) is recorded by the fix batch itself with batched follow-up and zero current exposure. Remaining known-unknowns are honestly deferred to Phases 6–7 and covered by later-phase success criteria (AUTH-02, re-checked at REQUIREMENTS.md:48/134; SESS-03). Status: **passed**.

---

_Verified: 2026-11-19T19:05:00Z_
_Verifier: Claude (gsd-verifier)_
