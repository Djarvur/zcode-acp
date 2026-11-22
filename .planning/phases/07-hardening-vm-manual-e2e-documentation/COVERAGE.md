# Phase 7 — Probe Ledger & Coverage Declarations

**Phase:** 07-hardening-vm-manual-e2e-documentation
**Declared:** 2026-11-22
**Kind:** specless probe ledger (finalized at plan time) + reasoned integration declarations

## API Coverage — reasoned declaration

**No external API integration: this phase integrates no new external API.** The
deterministic detector (`api-coverage.cjs --json`) returned `skipped` (no phase-scope
verb+noun signal), and the claim is true on the merits: Phase 7's work is session-compat
arms, dialect coverage, CI composition, a VM script, and documentation over ALREADY
integrated surfaces (the pinned acp-go-sdk v0.20.0 front leg, the existing api.z.ai PAT
chain from Phase 6, the desktop app-server wire from Phase 3). Zero new dependencies
(Package Legitimacy Audit: vacuously passes — no install tasks). No matrix rows to mint.

## Assumption-Delta Advisory

`assumption-delta scan "7"` returned `detected:false` (no pluralization/optional/chosen
signals in the phase section). Recorded for the audit trail; no action. The omp
subcommand is a new CLI surface, not a singular-to-plural migration of an existing
identity model.

## Specless Probe Fallback — Edge Ledger (finalized)

12 applicable probe items from the coverage report: **6 resolved explicit** (the
categorized edges — authored as plan `must_haves.truths` with named tests), **6
unclassified → carried as flagged assumptions** (never auto-resolved, never silently
dropped). Equality: **12 = 6 authored + 6 flagged**.

| # | Requirement | Probe (category) | Resolution | Plan/Task |
|---|-------------|------------------|------------|-----------|
| 1 | QA-01 | empty — empty/single/null input | A missing or skipped arc leg fails the named suite; a zero-frame run is a reported failure (the TestEOFSignature posture carried into the arc) | 07-06 T1 (per-leg frame asserts; no-skip gate) |
| 2 | QA-02 | concurrency — interrupted/parallel guarantee | Each script check prints a named PASS/FAIL line; the script exits non-zero naming the first failed check; re-run after interruption is safe (no destructive step without an explicit flag) | 07-07 T1 (check helper + summary contract), T3 (re-run safety) |
| 3 | SESS-02 | empty — empty/single/null input | session/delete on an unknown, already-closed, or EMPTY session id answers {} immediately with zero backend frames (the spec silent-success arm) | 07-03 T2 (TestSessionDeleteUnknownSilentSuccess — empty-id case included) |
| 4 | SESS-03 | concurrency — interrupted/parallel guarantee | A refused /model switch never holds the front turn slot (the intercept precedes acquireTurn); session/delete on a closing session rides the existing isClosing gate via the 07-01 teardown contract | 07-05 T3 (TestModelSlashRefusalHoldsNoTurnSlot), 07-03 T1 (delete reuses CloseSession) |
| 5 | SESS-03 | encoding — bytes/codepoints/normalized form | The /model selector resolves by byte-exact, case-sensitive equality against the listed config-option values — no normalization, no case folding; only the single separating space is consumed | 07-05 T3 (TestModelSlashSelectorByteEquality) |
| 6 | SESS-03 | empty — empty/single/null input | `/model` with no argument, or no model bound in the session, refuses with data.details starting the pinned `No model selected.` string | 07-05 T2 (TestModelSlashNoModelRefusal — both arms) |
| 7 | QA-02 | unclassified | FLAGGED: VM field behavior beyond the scripted assertions (desktop-layout variance on real hardware) is observed at the manual VM run — the script is the deliverable (D-05) | 07-07 (SUMMARY), field observation in 07-09 T2 human-check |
| 8 | QA-03 | unclassified | FLAGGED: manual E2E client-behavior variance (A1 --model id shape; A3 picker behavior with configOptions but no models field) — observed and recorded at the human gate, never assumed | 07-09 T3 (observation items in the human-check; backstop marker in must_haves) |
| 9 | SESS-03 | unclassified | FLAGGED: any method OUTSIDE the documented DISC-02 table arriving from future client builds answers the SDK/adapter default decline — reviewed at the sweep; the table is the v1 contract | 07-03 T2 (the decline rows lock the documented set; the SDK fallback covers the rest) |
| 10 | DOCS-01 | unclassified | FLAGGED: README completeness beyond D-07's enumerated sections is reviewed at verification (check-docs.sh + the QA-03 run's "could I follow this?" judgment) | 07-08 T1, validated against the QA-03 manual run |
| 11 | DOCS-02 | unclassified | FLAGGED: recipe completeness beyond the pinned Pattern-8 inputs (e.g. client-version drift changing the tokenizer rules) is reviewed at the QA-03 live run | 07-08 T2, confirmed by 07-09 T3 |
| 12 | DOCS-03 | unclassified | FLAGGED: protocol-notes completeness beyond the provenance statement + limitations content is reviewed at verification | 07-08 T3 |

## Prohibition Recall — kept items (Stage 2 precision)

6 kept prohibitions, authored descriptor-less into plan `must_haves.prohibitions`
(flagged-unverified disposition downstream — no check descriptors minted). Equality:
**6 surfaced = 6 authored + 0 flagged-assumption prohibitions**.

| # | Requirement | Statement (abridged) | Category | Plan |
|---|-------------|----------------------|----------|------|
| P1 | SESS-02 | MUST NOT advertise sessionCapabilities delete/fork/list/load/resume — advertising delete flips Orca into calling delete on its disposable listing sessions | transparency | 07-03 |
| P2 | SESS-03 | MUST NOT forward an unvalidated /model selector to the backend — validate-first against confirmed configOptions | safety | 07-05 |
| P3 | QA-01 | MUST NOT retire, weaken, or skip-lock existing corpus suites — D-04: the bar is additive | transparency | 07-06 |
| P4 | QA-03 | MUST NOT record a passing verdict without the user's actual manual run — human confirmation precedes any evidence commit | transparency | 07-09 |
| P5 | DOCS-01/02 + QA-02 script | MUST NOT instruct exporting/pasting credential material — env-var NAMES and mechanisms only, placeholder forms | privacy | 07-08 (+ 07-07 threat row) |
| P6 | DOCS-03 | MUST NOT omit the Apache-2.0 attribution for ported reference-adapter code | transparency/legal | 07-08 |

**Canon-referral drops (breadcrumbs, not minted):**
- Credential logging in new CLI/script paths — generic sensitive-data-in-logs hygiene →
  owned by the per-plan threat models (T-07-12/T-07-20/T-07-26) + the scan-secrets gate.
- Argv/command-injection hardening in the omp dispatch — canon injection class →
  owned by the threat model (T-07-11) + the validate-first posture; no bespoke mint.
- Supply-chain downloads in the VM script — canon supply-chain class → owned by
  T-07-19 + T-07-SC + the checksum-verification rule; no bespoke mint.
- Secret shapes in doc examples — owned by the scan-secrets/check-docs gates (T-07-23).
