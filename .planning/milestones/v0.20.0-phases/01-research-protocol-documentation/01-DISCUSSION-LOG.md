# Phase 1: Research & Protocol Documentation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-19
**Phase:** 1-Research & Protocol Documentation
**Areas discussed:** Ground-truth source hierarchy, Experiment capture & artifacts policy, Docs layout & granularity, Provenance label format, Experiment execution policy, Orca live-interaction scope
**Mode:** `--auto` (manager-driven autonomous run; user directive: proceed automatically, stop only when unavoidable). Every choice below is the auto-selected recommended default, logged for audit.

---

## Ground-truth source hierarchy

| Option | Description | Selected |
|--------|-------------|----------|
| Installed 3.18.2 bundle as V-SRC ground truth; GitHub 3.17.x tree corroborates with version labels | Matches "never document 3.17.x behavior as 3.18.2 truth" (SUMMARY.md pitfall guidance) | ✓ |
| Fetch a 3.18.2 tag from GitHub as primary | Availability of an exact tag unverified; bundle is the actually-shipped artifact | |
| 3.17.x source tree as primary | Explicitly warned against by research | |

**User's choice:** *(auto)* Bundle-first V-SRC
**Notes:** If a matching GitHub tag/commit exists it is cited additionally; drift baseline documented either way.

---

## Experiment capture & artifacts policy

| Option | Description | Selected |
|--------|-------------|----------|
| Commit scripts + scrubbed captures (`experiments/`, `experiments/captures/`) | Seeds Phase 3 golden fixtures; scrub-first discipline | ✓ |
| Conclusions only, nothing committed | Cheaper but loses replay value for fixtures | |
| Commit raw JSONL | Risk of secret leakage in a public repo | |

**User's choice:** *(auto)* Scripts + scrubbed captures
**Notes:** Raw unscrubbed captures stay outside the repo. Secret shapes to strip: JWT, `enc:v2:` blobs, PATs, account ids.

---

## Docs layout & granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Split per deliverable + `docs/protocol/README.md` index/legend | Each success criterion maps to one file; task-able per file | ✓ |
| Single `docs/protocol.md` | Simpler but monolithic; harder to evolve per-phase | |
| Wiki-style external docs | Against repo-docs-in-English constraint | |

**User's choice:** *(auto)* Split per deliverable
**Notes:** `docs/research/reference-adapters.md` lives under `docs/research/` per the success-criterion wording.

---

## Provenance label format

| Option | Description | Selected |
|--------|-------------|----------|
| Inline tags + per-doc Sources section + shared legend in README | Greppable, survives doc splitting; legend written once | ✓ |
| Footnote apparatus | Heavier to maintain across five docs | |
| Separate provenance appendix per doc | Duplicates the legend five times | |

**User's choice:** *(auto)* Inline tags `[V-SRC|V-EXP|V-RPT]`
**Notes:** `V-EXP` ids resolve to `experiments/captures/` filenames.

---

## Experiment execution policy

| Option | Description | Selected |
|--------|-------------|----------|
| Targeted experiments for open items only; reuse FINDINGS V-EXP evidence | Minimal quota cost; retires exactly the open gaps | ✓ |
| Full re-verification incl. real turns | Wasteful quota burn; FINDINGS already field-verified most | |
| No new experiments | Leaves success criteria 1 & 4 without in-repo experiment records | |

**User's choice:** *(auto)* Targeted only
**Notes:** The five targeted experiments are enumerated in CONTEXT.md D-05. Real-turn terminal-signal experiment only runs if local credentials exist; otherwise V-SRC-only + recorded limitation. CI never runs experiments.

---

## Orca live-interaction scope

| Option | Description | Selected |
|--------|-------------|----------|
| Static source + on-disk artifacts only; live run deferred to Phase 7 | Phase 1 verifies, not integrates | ✓ |
| Headless omp-equivalent probe now | Belongs with CLI work (Phase 2+) | |
| Full live card run now | Manual E2E is the Phase 7 done-criterion | |

**User's choice:** *(auto)* Static verification only
**Notes:** omp contract table spot-checked by re-reading unpacked AppImage chunks; `--session-dir` format and `modelDiscovery` kind `session` answered statically.

---

## Claude's Discretion

- Experiment script implementations, capture formats, doc prose/depth
- Fallbacks when credentials or a deb layout are unavailable (record, don't block)
- Priority order of V-SRC(parallel) re-verification (load-bearing first)

## Deferred Ideas

- None new; v2 backlog items (EXT-01, SESSH-01, EXT-04) remain in REQUIREMENTS.md
