---
phase: "1"
slug: "research-protocol-documentation"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-19"
---

# Phase 1 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Real ZCode desktop bundle → experiment scripts | Experiments spawn the shipped 3.18.2 app-server locally and record its wire traffic | JSONL wire traffic (may contain credential-shaped values) |
| Capture pipeline → repository | `experiments/scrub-capture.mjs` is the only path into `experiments/captures/`; `experiments/scan-secrets.sh` gates every commit | Scrubbed JSONL captures, provenance-tagged docs |
| Private machine ↔ committed files | Real machine locations and account material stay outside the repo; scripts receive paths as CLI arguments | Neutral-phrased docs, path-free scripts |
| Upstream reference repo → repository | Read-only live study of github.com/william0wang/zcode-acp (Apache-2.0) | Adopt/avoid findings + attribution; zero cloned files committed |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-1-01 | Information Disclosure | experiments/captures/*.jsonl (all five captures) | high | mitigate | Scrub-first pipeline: raw captures born outside the repo; scrub-capture.mjs is the only path into experiments/captures/; scan-secrets.sh gates every commit (value-shape regexes, .planning excluded) | closed |
| T-1-02 | Information Disclosure | docs/**, experiments/** (path shapes) | high | mitigate | Neutral-phrasing rule; check-docs.sh path-shape grep fails the build on user-home absolute paths; scripts receive machine paths as CLI args only | closed |
| T-1-03 | Elevation of Privilege (account state) | experiments/app-server-probe.mjs flows | high | mitigate | No flow ever POSTs api_keys or mints credentials; reuse existing key material only; acceptance gate greps the harness for the minting endpoint (count 0 — verified: 0 hits) | closed |
| T-1-04 | Denial of Service (own plan quota) | turn flow | medium | mitigate | Exactly one tiny turn planned, fixed one-line prompt, quiet-window bound on collection, ledger in experiments/README.md; degradation path avoids retries (actual: 3 tiny sends, 0 completed, ledgered honestly) | closed |
| T-1-05 | Information Disclosure | keyring / credential probing | low | accept | Phase 1 needs no decrypted values; no flow reads the keyring or runs secret-service lookups; credential keys enumerated by name only | closed |
| T-1-06 | Tampering | bundle copy probed by all experiments | medium | mitigate | E4 re-ran the byte-compare of the main cjs against its .orig sibling (pristine/identical ×2 in EXP-DISCOVERY-01); dialect doc provenance header cites it | closed |
| T-1-07 | Repudiation (stale evidence) | capture headers + doc provenance | medium | mitigate | Harness records builtinRevision ("44") + date in every capture header at capture time; docs record which bundle copy was probed | closed |
| T-1-08 | Tampering (evidence) | docs/protocol/app-server-dialect.md provenance | high | mitigate | D-01 hierarchy in doc header and every claim tag; 3.17.x citations version-pinned; drift-baseline section explicit; check-docs.sh enforces tag grammar + V-EXP resolution; Sources records re-derivable commands with observed counts | closed |
| T-1-09 | Repudiation (stale evidence) | docs/protocol/acp-client-dialect.md build claims | medium | mitigate | Fresh AppImage extract (never volatile mounts); every re-verified claim carries chunk filename + anchor (`chunk` field per literal in EXP-SESSIONDIR-01); unconfirmed claims stay V-RPT | closed |
| T-1-10 | Repudiation (untraceable origin) | docs/research/reference-adapters.md attribution | medium | mitigate | Apache-2.0 attribution section (upstream has no NOTICE → per-file license header + README attribution + tag record); license live-verified; nothing upstream committed (git ls-files gate exit=1 = clean) | closed |
| T-1-11 | Tampering (host state) | deb/rpm artifact inspection | medium | mitigate | Inspection-only commands (dpkg-deb -I/-c) on scratch-dir artifacts; never installed; URLs fetched at execution time, never hardcoded | closed |
| T-1-SC | Tampering | npm/pip/cargo installs | high | accept | Zero external packages installed in Phase 1 (node:test, grep, jq only; no package.json/go.mod manifests exist) — nothing to gate | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-1 | T-1-05 | Phase 1 requires no decrypted credential values; keyring never read; keys enumerated by name only | user directive (full autonomous run) | 2026-11-19 |
| AR-2 | T-1-SC | Phase 1 installs zero external packages — stdlib tooling only (node:test, grep, jq) | user directive (full autonomous run) | 2026-11-19 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-19 | 12 | 12 | 0 | orchestrator (secure-phase, ASVS L1 short-circuit: threats_open 0 + register authored at plan time + L1 grep-depth) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-19
