---
phase: "1"
slug: "research-protocol-documentation"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-19"
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Shell check scripts (no test framework exists yet; greenfield docs phase). Node available for helpers. |
| **Config file** | none — Wave 0 installed `experiments/check-docs.sh` + `experiments/scan-secrets.sh` (plan 01-01) |
| **Quick run command** | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` |
| **Full suite command** | `bash experiments/check-docs.sh --phase-gate && bash experiments/scan-secrets.sh && node --test experiments/` (the `git grep` V-EXP provenance audit is built into check-docs.sh check (b); `--phase-gate` additionally enforces the six-doc heading table) |
| **Estimated runtime** | ~5 seconds measured (check-docs + scan-secrets + 12 node tests) |

---

## Sampling Rate

- **After every task commit:** Run `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` — this is what the six executor summaries record as the per-commit gate run (patterns-established in every SUMMARY; corroborated by 01-02, where scan-secrets demonstrably FAILED pre-commit on a real credential-residue defect before the fix landed)
- **After every plan wave:** Run full suite + human skim of changed doc sections (provenance tags are judgment-bearing)
- **Before `/gsd:verify-work`:** Full suite must be green; all six docs pass `--phase-gate` (verified exit 0 on 2026-11-19); every V-EXP id resolves with its capture header intact; secret-scan clean
- **Max feedback latency:** 30 seconds (measured: ~5 s)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-T1 | 01 | 1 | D-02/D-04 (CONTEXT) | T-1-01, T-1-02 | no secret shapes; legend defined once; path-shape grep; --phase-gate honest about missing docs | structure+gate check | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | ✅ | ✅ green |
| 01-01-T2 | 01 | 1 | D-02 (CONTEXT) | T-1-01 | scrubber rules tested (RED before GREEN; RED evidence 01-01-red-evidence.json) | unit tests (node --test) | `node --test experiments/` (12/12) | ✅ | ✅ green |
| 01-01-T3 | 01 | 1 | DISC-04 | T-1-01, T-1-07 | V-EXP id resolves + capture header records revision (header continuity now also gated by check-docs (b)/(f)) | tag+gate check | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh && grep -c '"capture":"EXP-WORKSPACEKEY-01"' experiments/captures/EXP-WORKSPACEKEY-01.jsonl` | ✅ | ✅ green |
| 01-02-T1 | 02 | 2 | DISC-01 | T-1-01, T-1-07 | capture verdict fields (accepted/refused per method, header intact) | capture header+gate | `grep -cE 'accepted\|refused' experiments/captures/EXP-SUBSCRIBE-01.jsonl && bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | ✅ | ✅ green |
| 01-02-T2 | 02 | 2 | DISC-01 | T-1-03, T-1-04 | one tiny turn max (ledgered honestly: 3 sends issued, 0 completed); honest degradation; no key-minting endpoint | capture header+gate | `grep -ciE 'api_keys' experiments/app-server-probe.mjs` (expect 0; now also check-docs (g)) `&& grep -c 'real turns spent' experiments/README.md && bash experiments/scan-secrets.sh` | ✅ | ✅ green |
| 01-03-T1 | 03 | 3 | DISC-01 | T-1-08 | tag grammar; capture ids cited and resolving (EXP-SUBSCRIBE-01 ×22, EXP-TURN-01 ×30) | structure+tag check | `bash experiments/check-docs.sh && grep -c 'V-EXP EXP-SUBSCRIBE-01' docs/protocol/app-server-dialect.md` | ✅ | ✅ green |
| 01-03-T2 | 03 | 3 | DISC-01 | T-1-08 | drift baseline first-class; 3.17.x version-pinned (3.17.1 ×36); V-RPT visible (×19); phase-gate per-doc pending 0 | structure+tag check | `bash experiments/check-docs.sh --phase-gate 2>&1 \| grep -c 'PENDING: docs/protocol/app-server-dialect.md'` (expect 0) | ✅ | ✅ green |
| 01-04-T1 | 04 | 2 | DISC-02 | T-1-09 | chunk citations; client build named in capture header (clientBuild ×1); floor string 17.0.5 re-verified (×3) | capture header+gate | `grep -c '"capture":"EXP-SESSIONDIR-01"' experiments/captures/EXP-SESSIONDIR-01.jsonl && bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | ✅ | ✅ green |
| 01-04-T2 | 04 | 2 | DISC-02 | T-1-09 | neutral phrasing; known-unknowns stay V-RPT (×11); V-EXP EXP-SESSIONDIR-01 cited ×5 | structure+tag check | `bash experiments/check-docs.sh && grep -c 'V-EXP EXP-SESSIONDIR-01' docs/protocol/acp-client-dialect.md` | ✅ | ✅ green |
| 01-05-T1 | 05 | 2 | DISC-03 | T-1-06, T-1-11 | no package installs; pristine byte-compare recorded (×2); artifact inspection via dpkg-deb (×4) | capture header+gate | `grep -ciE 'pristine\|identical' experiments/captures/EXP-DISCOVERY-01.jsonl && bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | ✅ | ✅ green |
| 01-05-T2 | 05 | 2 | DISC-03 | T-1-06 | matrix rows cite recorded commands (V-EXP EXP-DISCOVERY-01 ×16); running-mount do-not-rely (×3) | structure check | `bash experiments/check-docs.sh && grep -c 'V-EXP EXP-DISCOVERY-01' docs/protocol/install-discovery.md` | ✅ | ✅ green |
| 01-06-T1 | 06 | 2 | DISC-05 | T-1-10 | license live-verified (re-verified 2026-11-19: Apache-2.0, exit 0); nothing upstream committed (`git ls-files --error-unmatch` exit 1) | CLI check | `gh api repos/william0wang/zcode-acp --jq '.license.spdx_id'` | n/a (network) | ✅ green |
| 01-06-T2 | 06 | 2 | DISC-05 | T-1-10 | Apache-2.0 attribution section present (×10); seven headings; phase-gate per-doc pending 0 | structure check | `bash experiments/check-docs.sh --phase-gate 2>&1 \| grep -c 'PENDING: docs/research/reference-adapters.md'` (expect 0) | ✅ | ✅ green |

*File Exists: all Wave-0 infrastructure and all five captures now exist (verified on disk 2026-11-19). 01-06-T1 is a network check by design (GitHub API), with the `git ls-remote` degraded fallback documented in the plan.*

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky — every row re-executed 2026-11-19 against HEAD and green. Count columns are the observed grep counts at audit time.*

---

## Wave 0 Requirements

- [x] `experiments/check-docs.sh` — structure + provenance-tag conformance for all six docs (exists; extended 2026-11-19 with capture-header continuity (b)/(f) and no-key-minting (g) — see audit note)
- [x] `experiments/scan-secrets.sh` — secret-shape gate (JWT / `enc:v2:` / PAT regexes; Phase 2 CI reuses it)
- [x] `experiments/scrub-capture.mjs` — canonical scrubber (existed BEFORE the first capture was written; 12 unit tests green)
- [x] `experiments/README.md` — how to run E1–E5; scrub-first rules; "never in CI" warning; quota ledger
- [x] `docs/protocol/README.md` — index + provenance legend (legend defined exactly once, enforced by check-docs (a))

All five verified present and the full triple `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh && node --test experiments/` exits 0 (2026-11-19).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Bundle-derived claims re-derivable (spot-check method strings via grep) | DISC-01 | Bundle path is machine-private (supplied in agent prompts, never committed) | `grep -o 'sessionSubscribe:"session/subscribe"' <bundle>` — record result in the doc |

Plus the judgment-tier human skims documented in the six SUMMARY `coverage.human_judgment` entries (provenance-tag honesty per T-1-08/T-1-09/T-1-10; quota-ledger framing; ops-guide accuracy) — performed at `/gsd:verify-work` by design, not reducible to mechanical gates.

---

## Nyquist Audit (2026-11-19)

Audited against all six plans' `<verify>` blocks and the six executor summaries:

1. **Automated verify coverage: 13/13 tasks** — every task in plans 01-01…01-06 carries `<automated>` verify commands; no task relies on a Wave-0 dependency alone. The only manual row is the documented bundle spot-check above.
2. **Sampling continuity: satisfied** — 13/13 tasks automated, so no window of 3 consecutive tasks without automated verify exists (trivially, none without).
3. **Map ↔ plans consistency: repaired** — task IDs already matched all 13 plan tasks; two rows were inaccurate and fixed: 01-02-T1 previously claimed "no key-minting endpoint" (that gate belongs to 01-02-T2) and several rows listed only generic gate commands without the plan's behavioral greps. Commands now reflect the real task verify blocks. Statuses were all ⬜ pending and are now set from re-execution.
4. **Quick-run command vs actual practice: confirmed** — `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` is exactly what every summary records as the per-commit gate; the claim is corroborated by 01-02's pre-commit scan-secrets FAILURE catching a real credential-residue defect. All 21 summary-claimed production/metadata commits verified as ancestors of HEAD (one self-check cited commit `aaa258f`, which was later amended into `da35cdb` — the content commit `e7c7745` is intact; bookkeeping artifact only).
5. **Gaps found and filled (in `experiments/check-docs.sh`; exit semantics unchanged — violations print `CHECK FAIL:` and exit 1):**
   - **Capture-header continuity (T-1-07):** check (b) resolved V-EXP ids but never verified the capture still carries its `{"capture":"<id>"}` header line — a re-scrub or edit stripping the header would keep every gate green while silently weakening the stale-evidence guard. Now every resolved capture file's first line must contain its id. Red-proven: an injected headerless sibling capture fails the gate.
   - **No-key-minting continuity (T-1-03):** the `api_keys` absence gate ran only at plan 01-02 task time; the harness is a living file later phases edit. Now a permanent check over `experiments/*.mjs` + `*.sh` (check-docs.sh excludes itself — it must name the pattern it enforces). Red-proven: an injected `api_keys` reference in an experiments script fails the gate.
   Both checks were green on first full run and re-verified red-able by mutation probes the same day.
6. **Frontmatter honesty:** `wave_0_complete: true` (all five artifacts exist, full triple exit 0), `status: validated`, `nyquist_compliant: true` (13/13 automated, continuity holds, map matches plans, quick-run verified, all rows re-executed green).

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (13/13 automated)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s (measured ~5 s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-11-19 (nyquist audit — all map rows re-executed green at HEAD; two continuous-gate gaps filled and red-proven; `check-docs.sh --phase-gate` exits 0 with all six deliverable docs present)
