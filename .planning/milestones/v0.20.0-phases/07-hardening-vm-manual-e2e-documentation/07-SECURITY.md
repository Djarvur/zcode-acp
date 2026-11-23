---
phase: "07"
slug: "hardening-vm-manual-e2e-documentation"
status: final
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-22"
---

# Phase 07 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| wrapper ↔ ACP client | JSON-RPC 2.0 stdio (omp/acp CLI dispatch; argv parsed by stdlib FlagSet) | session stream, configOptions, /model slash refusals (code -32603 + data.details), --model launch binding |
| wrapper ↔ desktop app-server | JSONL over stdio (spawned child; paired provider-config env) | protocol frames, account snapshot, lifecycle events |
| wrapper ↔ keyring | `secret-tool` subprocess (bounded 5 s probe) | coded classification only (arm names + latency, never key material) |
| VM scenario ↔ clean machine | `scripts/vm-e2e.sh` — presence checks with hints only; never installs, never downloads, never pipes network into shell | version floor, sentinel byte-matches, read-back facts |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-07-01 | Tampering | acpfront/lifecycle | medium | mitigate | `registerIfOpen` holds closingMu across re-check + release + Register (lifecycle.go:322-340); TestResumeCloseInterleave green under -race | closed |
| T-07-02 | DoS | acpfront/front | medium | mitigate | `teardownAfterPickerRefusal` → full CloseSession; wired at picker + slash arms; obsolete wire-only call grep = 0 | closed |
| T-07-03 | DoS | acpfront | low | mitigate | closeOrphan drops cwd+settings before the wire; resume refusal drops both; dead guard removed | closed |
| T-07-04 | DoS | acpfront | low | accept | Accepted Risks Log AR-07-01 (close-toast 10 s budget, README:93-94) | closed (accepted) |
| T-07-05 | Tampering | hostsvc/snapshot | medium | mitigate | single ReadFile feeds both revision and rules (decodeBuiltinTable); readBuiltinProviderRules grep = 0 | closed |
| T-07-06 | DoS | interact/keyringprobe | low | mitigate | classify order: parent-cancel → success → deadline → marker → default; boundary test green | closed |
| T-07-07 | Tampering | interact/keyringprobe | medium | mitigate | sentinel block byte-intact; script table byte-identical (sort -u = 1 per sentence) | closed |
| T-07-08 | Tampering | acpport/agent | high | mitigate | SessionCapabilities assembly exactly once, Options-only; advertisement-honesty guard test (mutation-verified) | closed |
| T-07-09 | DoS | acpfront/front | low | accept | Accepted Risks Log AR-07-02 (DeleteSession reuses the CloseSession cancel-drain; silent-success window bounded) | closed (accepted) |
| T-07-10 | Tampering | acpport/agent | medium | mitigate | explicit UnstableForkSession decline arm; TestSessionForkDeclineByteLocked | closed |
| T-07-11 | Tampering | cmd/main | medium | mitigate | ompFlagSet dispatch + stamp 18.1.9+zacp.0; TestOmpLaunchBinding binary E2E | closed |
| T-07-12 | Information disclosure | cmd/main | medium | mitigate | binding log carries ids + levelExplicit bool only; scan-secrets green | closed |
| T-07-13 | Repudiation | cmd/main | low | accept | Accepted Risks Log AR-07-03 (version stamp 18.1.9+zacp.0 = the honest self-stamp; README floor framing true post-07-05) | closed (accepted) |
| T-07-14 | Tampering | acpfront/front | high | mitigate | /model validate-first against confirmed configOptions values, refusal BEFORE the backend setModel; WR-02 busy gate under turnMu | closed |
| T-07-15 | Tampering | acpport | medium | mitigate | Details field + NewInternalWithData; wireError injects data.details only when non-empty; pinned strings single-site | closed |
| T-07-16 | DoS | acpfront/front | low | accept | Accepted Risks Log AR-07-04 (the /model intercept precedes acquireTurn by design — refusals are slot-free; the turn-active gate added by WR-02 bounds the mutation case) | closed (accepted) |
| T-07-17 | Repudiation | acpfront/conformance | medium | mitigate | TestConformanceV1Arc green, zero t.Skip; additive-only (34-scenario golden anchor untouched) | closed |
| T-07-18 | DoS | acpfront/conformance | low | mitigate | conformance suite green under -race -count=3 | closed |
| T-07-19 | Elevation | scripts/vm-e2e.sh | high | mitigate | no download/install mechanism (no curl/wget/npm/pip/apt); binary from --binary argv; header documents build-or-fetch + sha512 precedent | closed |
| T-07-20 | Information disclosure | scripts/vm-e2e.sh | medium | mitigate | env-var NAMES only; SKIP messages state "value never echoed"; scan-secrets green | closed |
| T-07-21 | DoS | scripts/vm-e2e.sh | low | mitigate | timeout(1) wraps the internal arm; probe 5 s bound intact | closed |
| T-07-22 | Information disclosure | README + provenance | medium | mitigate | placeholder-only (shim sources a user-owned env file); check-docs + scan-secrets green | closed |
| T-07-23 | Information disclosure | README + provenance | medium | mitigate | no user-home absolute paths or credential shapes; both gates green | closed |
| T-07-24 | Repudiation | README + provenance | low | mitigate | doc claims read back against code (refusal strings, sentinels, close budget, version floor) | closed |
| T-07-25 | Repudiation | UAT + experiments ledger | high | mitigate | honest-recording artifacts: UAT Test 2 awaiting (never flipped without the user's Orca run); Test 1 PASSED records a REAL VM run with delegation provenance; ledger carries gap-cycle spend honesty | closed |
| T-07-26 | Information disclosure | experiments ledger | medium | mitigate | scrub-first + scan-secrets-on-tree-and-staged-diff discipline | closed |
| T-07-SC | Tampering (supply chain) | go.mod | high | mitigate | exactly 1 require line (acp-go-sdk v0.20.0); vm-e2e.sh does presence checks with hints only | closed |

*Status values: closed / open — below high threshold (non-blocking).*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open.*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party).*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-07-01 | T-07-04 | The close toast budget is 10 s by design (lifecycle.go:50, README:93-94); slower backends surface the budget rather than hang the client. | orchestrator (auto, ASVS L1 low) | 2026-11-22 |
| AR-07-02 | T-07-09 | DeleteSession's silent-success window (front.go:640-649) is bounded by the CloseSession cancel-drain it reuses; the delete no-op is a compat surface, not a data path. | orchestrator (auto, ASVS L1 low) | 2026-11-22 |
| AR-07-03 | T-07-13 | The version stamp `18.1.9+zacp.0` is the honest self-stamp (the card floor is 17.0.5; 18.1.9 resolves selectors); -ldflags seam preserved for release builds. | orchestrator (auto, ASVS L1 low) | 2026-11-22 |
| AR-07-04 | T-07-16 | The /model intercept precedes acquireTurn by design: refusals are slot-free so a mistyped model never burns a turn; WR-02 added the turn-active gate for the mutation case. | orchestrator (auto, ASVS L1 low) | 2026-11-22 |

---

## Surface Inventory (post-plan additions)

- The `/model` slash-prompt layer (front.go promptModelSlash): validate-first against confirmed configOptions; refusals ride code -32603 with `data.details` (the detail seam, opt-in — generic frames keep data null, guard-tested); Hb strings byte-pinned single-site.
- The omp CLI dispatch + `--model` launch binding + version floor (cmd/main.go): stdlib FlagSet (zero new deps, tidy-diff gate asserted); the 18.1.9+zacp.0 self-stamp (AR-07-03).
- The vm-e2e.sh no-login-arm recognition (8da356b): audited per the 8da356b review note — under `--require-auth` the no-desktop-login arm hard-fails; the default pass names the observed arm; any other coded error still fails the stage. The patch does NOT weaken the hardening.

**Provenance note (T-07-25):** the QA-02 PASSED verdict records an agent-executed VM run made under the user's explicit recorded directive ("организуй ее сам" — organize the VM yourself, 2026-11-22); the run was real and honestly recorded (KVM clean Ubuntu 24.04.5 VM, 16 checks / 1 skip, exit 0). QA-03's user gate remains open pending the user's actual Orca run — flipping it without that run would violate T-07-25's own mitigation.

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-22 | 27 | 23 | 4 (low, accepted → AR-07-01..04) | gsd-security-auditor (ASVS L1, mitigate-verification mode; every mitigation traced to code line, every test lock run live under -race; the 4 accept rows close mechanically with the entries above) |

**Notes:** the phase-7 review (07-REVIEW.md: 0 critical / 3 warnings / 8 info) was fixed-in-phase for WR-01/02/03 (06-REVIEW-DISPOSITION-equivalent at 07-REVIEW-DISPOSITION.md); the 8 info advisories route post-v1. The inherited phase-6 security posture (PAT memory-only, keyring sentinels, registry-as-verdict) was re-verified where phase 7 touches those surfaces.
