---
phase: "3"
slug: "zcode-back-leg-codec-fake-app-server-supervisor"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-20"
---

# Phase 3 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Child process ↔ codec | stdio pipes carry protocol frames ONLY (stdout discipline: no diagnostics on stdout; logs → stderr slog) | JSONL frames; oversized/malformed lines skipped+logged, stream continues |
| Discovery probes → filesystem | Probe order first-match-wins; volatile /tmp/.mount_* decoys recorded-and-rejected; localhost listeners never probed | Install candidates (path, revision) |
| Supervisor → child env | Explicit allowlist (PATH, HOME + ZCode vars); proxy variables scrubbed | Child's OBSERVED environment asserted by test |
| CI → real desktop | Build tag `realdesktop` + env guard `ZCODE_ACP_REAL_E2E=1` — desktop unreachable from CI by construction | Nothing (guarded skip proven machine-checkable) |
| Go module supply chain | Zero new dependencies (stdlib only); `mise run tidy-diff` fails on any go.mod/go.sum drift | go.mod module-line-only at HEAD |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-3-01 | Tampering | codec stdout/stdin discipline | high | mitigate | Library never touches os.Stdout (grep: 4 hits all in fake/stdio.go where stdout IS the protocol channel — by design); diagnostics → stderr slog; golden tests assert stdout=frames-only | closed |
| T-3-02 | Denial of Service (resource) | conn read loop line buffer | medium | mitigate | Raised-but-capped buffer (initial 1 MiB, max 16 MiB, conn.go:169); oversized line skipped+logged, stream continues | closed |
| T-3-03 | Tampering (correlation integrity) | conn demux + pending table | high | mitigate | WR-02 structural rule test-locked (method-bearing frames never resolve pending calls; ids correlate by value AND type) — the 02-REVIEW WR-02 false-accept cannot recur silently (TestDemux green) | closed |
| T-3-04 | Tampering | fake scenario loader + fixtures | low | mitigate | Fail-loud validation naming the field (03-03 TDD); scenarios repo-controlled, never user input; fixed synthetic ids | closed |
| T-3-05 | Tampering/Spoofing (decoys) | discovery probes | medium | mitigate | /tmp/.mount_* recorded-and-rejected (test-locked); first-match-wins per documented order (TestDiscover green) | closed |
| T-3-06 | Tampering (state integrity) | turn observe/classify path | medium | mitigate | Single observe path, turnId-matched carrier-tolerant classification, exactly-once resolution test-locked; no synthesized completion on absence of evidence (prohibition) | closed |
| T-3-07 | Denial of Service (resource) | fake child lifecycle in tests | medium | mitigate | Setpgid + reap; exit-0-with-zero-lines = failure signature (errZeroFrameExchange); silent-dead child cannot read as pass | closed |
| T-3-08 | Tampering (env injection) | supervisor ChildEnv | high | mitigate | Explicit allowlist + proxy scrub, asserted on child's OBSERVED environment (TestSpawnEnv green) | closed |
| T-3-09 | Denial of Service (lifecycle) | staged shutdown + group kill | high | mitigate | Setpgid spawn, staged shutdown, group SIGKILL, ESRCH-verified zero survivors incl. grandchildren; crash-window 3/60s prevents churn (TestGroupKill green) | closed |
| T-3-10 | Tampering (CI isolation) | realdesktop_test.go | high | mitigate | Build tag + env guard; default suite green with the file present; real desktop unreachable from CI by construction | closed |
| T-3-11 | DoS (hang)/Repudiation (double answer) | turn cancel path | high | mitigate | Exactly-once cancelled resolution (PITFALL 4); stop travels bypass lane — cancel cannot deadlock behind prompts; idempotent repeats (TestCancel green) | closed |
| T-3-12 | Information Disclosure (credential material) | credential-cipher handling | high | mitigate | Phase 3 NEVER answers interaction/requestCredentialCipher — tracked-no-answer with recorded expiry, turn-tier honesty test; zero fabricated/forwarded credential material | closed |
| T-3-13 | Tampering (CI isolation) | e2e_test.go + default suite | high | mitigate | E2E child = built FAKE binary only (34 refs); real-desktop behind 03-07's tag+env guard; CI never contacts api.z.ai or the desktop | closed |
| T-3-SC | Tampering | Go module installs | high | mitigate | Zero new module dependencies — go.mod module-line-only at HEAD (stdlib only; goldie excluded per STACK.md); tidy-diff gate re-proven every task | closed |

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
| 2026-11-20 | 14 (13 unique + SC) | 14 | 0 | orchestrator (secure-phase, ASVS L1 short-circuit: threats_open 0 + register authored at plan time + L1 grep/live-test verification incl. real-desktop guarded run) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-20
