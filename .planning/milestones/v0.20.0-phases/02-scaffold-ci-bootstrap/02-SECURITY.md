---
phase: "2"
slug: "scaffold-ci-bootstrap"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-19"
---

# Phase 2 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| GitHub Actions runners → repository | CI executes workflow shell steps on every push/PR | Event metadata (no untrusted interpolation), toolchain from official channels |
| Toolchain installs | mise-managed (go, golangci-lint, goreleaser, node) from official distribution channels; exact pins | Versioned binaries — substitution detectable via version gates |
| GitHub API (protection PUT) | Single authenticated admin call with the reviewed versioned payload | protection-body.json (reviewed in-repo before apply) |
| master merge gate | Required checks + PR-only + enforce_admins | Merges only through green `build / vet / lint / test / tidy` + `govulncheck` + `secret-scan` |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-2-01 | Information disclosure | secret-gate + scrubber (the capture pipeline → required CI check) | high | mitigate | WR-03 boundary-aware rewrite + IN-04 header guard (02-02, TDD RED→GREEN); WR-01 scan-gate class aligned byte-identical to scrubber CRED_RX (02-03); gate promoted to required CI check (02-04, observed green both legs) and required protection context (02-05, GET-verified). REVIEW CR-01 re-opened it (USER_INFO_RX lastIndex leak on multi-object lines — reproduced); fixed TDD (commits 3969eae/4ed831a, multi-object + fixed-point regressions, suite 20/20) — re-closed | closed |
| T-2-02 | Tampering | master branch history (force-push, deletion) | medium | mitigate | `allow_force_pushes: false`, `allow_deletions: false`, `required_linear_history: true` — GET-verified live | closed |
| T-2-03 | Elevation of privilege | direct pushes to master bypassing review | medium | mitigate | `required_pull_request_reviews` + `enforce_admins: true`; proven empirically by the GH006 negative push test (remote rejected: "Changes must be made through a pull request") | closed |
| T-2-04 | Tampering | workflow shell steps (event-field interpolation) | low | mitigate | Zero untrusted event fields interpolated (grep: 0 hits outside event_name/ref in concurrency); goswitch env-indirection discipline preserved in e2e-matrix | closed |
| T-2-05 | Elevation of privilege | GITHUB_TOKEN scope per workflow | medium | mitigate | Explicit `permissions:` in all four workflows; `contents: read` everywhere except release.yml `contents: write` (cross-file gate) | closed |
| T-2-SC | Tampering | package/toolchain installs (mise tools, actions, npm) | high | mitigate | Zero registry packages (node:test built-in; toolchains via mise official channels, exact pins D-02; single action `actions/checkout@v4`; govulncheck first-party from source); RESEARCH Package Legitimacy Audit: all rows OK/Approved, no [ASSUMED]/[SUS] | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| (none) | — | All threats mitigated; no accepted risks this phase | — | — |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-19 | 6 | 6 | 0 | orchestrator (secure-phase, ASVS L1 short-circuit: threats_open 0 + register authored at plan time + L1 grep/GET verification) |
| 2026-11-19 | 6 | 6 | 0 | code-review re-audit: CR-01 re-opened T-2-01 (scrubber lastIndex PII leak, reproduced); fixed TDD RED→GREEN (3969eae/4ed831a), WR-01 job timeouts added (3c13b6b); re-closed, threats_open stays 0 |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-19
