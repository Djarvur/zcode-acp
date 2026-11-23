---
phase: "4"
slug: "acp-front-minimal-chat-loop"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-20"
---

# Phase 4 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| ACP client ↔ acpfront | SDK-framed JSON-RPC over stdio; capabilities advertised = exactly what exists (zero-value AgentCapabilities — no fs/terminal fields by construction) | protocol frames; honest initialize/authenticate |
| Front ↔ back leg | internal/acpport seam (only non-test SDK importer — boundary test enforced) | typed translations; no raw SDK leakage |
| cmd/zcode-acp → child | supervisor Override seam (explicit `--app-server`/`ZCODE_ACP_APP_SERVER` operator/test flag, default = real discovery) | spawn env allowlist (Phase 3) |
| Go module supply chain | First approved dependency: coder/acp-go-sdk v0.20.0 exact pin (D-01 audit PASSED: Apache-2.0, coder org, zero transitive deps); tidy-diff gate asserts the expected require set | go.mod one-line |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-4-01 | Tampering | Unvalidated/relative cwd reaching backend session/create | medium | mitigate | Absolute-cwd check before ANY backend call; negative test asserts no create frame on the wire (TestNewSessionRejectsRelativeCwd green) | closed |
| T-4-02 | Information disclosure | Prompt content/params on error/log paths (new packages + cmd) | medium | mitigate | House rule: slog to stderr, structural fields only (method, session id, id keys, sizes) — never params/content; grep-audited (sole hit = SDK interface signature, not a log) | closed |
| T-4-03 | Denial of Service | Inbound frame/queue floods at the ACP boundary | low | accept | SDK bounded queue (1024) + 10 MiB scanner; codec 16 MiB cap; per-envelope synchronous delivery = designed backpressure; Phase 4 payloads small | closed |
| T-4-04 | Tampering | Dishonest capability report / fake auth success | medium | mitigate | initialize byte-locks the honest probe shape on the RAW frame (zero fs/terminal/loadSession); authenticate answers honest not-available (-32603); advertise=implemented prohibition in must_haves | closed |
| T-4-05 | DoS / state corruption | Concurrent second prompt bypassing the turn guard | medium | mitigate | Both guard layers map to ONE invalid-request surface (front slot + machine ErrTurnActive numeric -32010); test-locked (TestSecondPrompt green) | closed |
| T-4-SC | Tampering | SDK supply chain (the first approved dependency) | high | mitigate | D-01 Package Legitimacy Audit PASSED every axis (Apache-2.0, coder org, zero transitive deps, active maintenance); exact pin v0.20.0; tidy-diff gate asserts the expected require set (any unapproved addition goes red); TestSDKImportBoundary: zero non-test SDK imports outside internal/acpport | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-1 | T-4-03 | Structural queue/scanner caps documented (SDK 1024 queue, 10 MiB scanner; codec 16 MiB); Phase-4 payloads small; synchronous delivery is the designed backpressure | standing directive (full autonomous run) | 2026-11-20 |
| AR-2 | T-4-05 (partial) | Supervisor Override seam accepts explicit operator/test flag for arbitrary executables — same trust position as Phase 3's defaultOverride seam; defaults to real discovery | standing directive (full autonomous run) | 2026-11-20 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-20 | 6 (5 unique + SC) | 6 | 0 | orchestrator (secure-phase, ASVS L1 short-circuit: threats_open 0 + register authored at plan time + L1 grep/live-test verification) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-20
