# Phase 7: Hardening, VM & Manual E2E, Documentation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-22
**Phase:** 7-Hardening, VM & Manual E2E, Documentation
**Mode:** `--auto` — all gray areas auto-selected, recommended defaults logged below

---

## [auto] Gray-area selection

[--auto] Selected all gray areas: Session-compat surface (SESS-02), Full Orca dialect coverage (SESS-03), CI protocol conformance (QA-01), Reproducible VM scenario (QA-02), Manual E2E (QA-03 — USER GATE), Documentation (DOCS-01/02/03), Routed hardening (WR/IN from 06-REVIEW-DISPOSITION.md).

## Session-compat surface (SESS-02)

[auto] Q: "What shape do session/delete and session/fork answers take?" → Selected: "Dialect-verified graceful shapes — delete as a graceful no-op (close semantics where applicable), fork handled gracefully; researcher pins exact result schemas; fake corpus byte-locks both" (recommended default)

## Full Orca dialect coverage (SESS-03)

[auto] Q: "How is the DISC-02 method sweep proven?" → Selected: "One-by-one sweep — implemented or gracefully handled, proven by a fake-corpus conformance row per uncovered method" (recommended default)

[auto] Q: "How does the persona CLI contract land?" → Selected: "--model launch binding rides the existing setModel follow-up machinery; /model slash prompt with the exact Hb strings byte-locked; launchesWithModel:true claimable in DOCS-02" (recommended default)

## CI protocol conformance (QA-01)

[auto] Q: "What is the named conformance suite?" → Selected: "One composed scenario over the existing proven corpus pieces — initialize → new → streaming → tools → permission → cancel — byte-locked, green in mise run ci" (recommended default)

## Reproducible VM scenario (QA-02)

[auto] Q: "What is the VM deliverable?" → Selected: "Scripted in-repo (install → configure both paths → run → no-keyring error path); running on an actual VM is the user's manual act, recorded in the ledger" (recommended default)

## Manual E2E (QA-03 — USER GATE)

[auto] Q: "Who performs the Orca card E2E?" → Selected: "The user, on the real desktop, both auth paths; wrapper side completes everything automatable; recorded via UAT + ledger (phase-6 precedent); end-of-phase human gate, never autonomous" (recommended default)

## Documentation (DOCS-01/02/03)

[auto] Q: "Language and layout of the user surface?" → Selected: "English README + preset-agent launch recipe + docs/ protocol notes with provenance (Apache-2.0 attribution, no-patching statement) and known limitations (no OAuth quota counter; Start Plan not headless)" (recommended default)

## Routed hardening (WR/IN)

[auto] Q: "Where do WR-01/04/05 and IN-01..04 land?" → Selected: "One lifecycle-consistency pass for WR-01/04/05 + batched advisory cleanups for IN-01..04, in this phase (their designed home per 06-REVIEW-DISPOSITION.md)" (recommended default)

---

## Claude's Discretion

- Exact delete/fork result shapes (dialect-verified); conformance scenario naming/location; VM script form; docs/ layout; IN-03 disposition (behavior vs comment); wave decomposition

## Deferred Ideas

- session/load/fork real support (v2); elicitation preference (beyond v1); non-Ubuntu-GNOME (accepted PRs post-v1)
