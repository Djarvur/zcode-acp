---
phase: "06"
slug: "oauth-coding-plan-path-model-picker"
status: final
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-22"
---

# Phase 06 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| wrapper ↔ desktop app-server | JSONL over stdio (the spawned child) | protocol frames, account snapshot (entitlement facts), credential-cipher asks (enc blobs + decrypted values in-reply scope only) |
| wrapper ↔ api.z.ai | HTTPS (the PAT minter; proxy-aware, scrubbed env) | OAuth login token (from the desktop's own credential store), minted PAT (in-process only, never persisted) |
| wrapper ↔ GNOME Keyring | `secret-tool` subprocess (the D-06 KeyLoader seam) | login-keyring key derivation input (sha256 of the resolved credentials path — never the key bytes on argv) |
| wrapper ↔ ACP client | JSON-RPC 2.0 (the front leg) | session stream, permission asks, configOptions — no credentials cross this boundary |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-06-SC | Tampering (supply chain) | go.mod | high | mitigate | single require (`acp-go-sdk v0.20.0`, pre-pinned); zero new dependencies this phase | closed |
| T-06-01 | Tampering/InfoDisc | interact/credstore | high | mitigate | unhandled `enc:` prefixes skip with structural log — never served as plaintext (credstore.go:317-323; test-locked) | closed |
| T-06-02 | Information Disclosure | interact/credstore | high | mitigate | logs carry key name/value_len/codes only (credstore.go:143-187; TestCredStoreLogHygiene) | closed |
| T-06-03 | Tampering | interact/cipherdecrypt | medium | mitigate | kid mismatch checked BEFORE GCM open (cipherdecrypt.go:227-230; TestCredentialDecryptKeyMismatch) | closed |
| T-06-04 | Information Disclosure | hostsvc/patminter | high | mitigate | zero write constructs; no String/Marshal on Pat; TestPatMemoryOnly AST lock | closed |
| T-06-05 | Tampering/InfoDisc | hostsvc/patminter | high | mitigate | pinned origin `https://api.z.ai`; transport cloned + proxy resolved/logged once; 15 s per-call timeout | closed |
| T-06-06 | EoP (stale PAT) | hostsvc/patminter | high | mitigate | Pat.AccountScope binding; Invalidate/Cached/Ensure single-flight (patminter.go:226-228,424,499,513) | closed |
| T-06-07 | DoS (duplicate zcode-api-key) | hostsvc/patminter | low | accept | process-local race, cosmetic, self-healing via list-first search (PLAN 06-02 rationale) — Accepted Risks Log below | closed (accepted) |
| T-06-08 | Tampering | hostsvc/snapshot | medium | mitigate | basedOn revision computed over the runtime-resolved path; path sensitivity test-locked | closed |
| T-06-09 | Tampering | hostsvc/snapshot | medium | mitigate | overlay entries from rule fields only; gate omission-only; zero-rules coded error | closed |
| T-06-10 | Tampering | acpfront/configoptions | medium | mitigate | translator inputs only reply-dug Settings fields (TestConfigOptionsTranslator) | closed |
| T-06-11 | Tampering | acpfront/configoptions | medium | mitigate | currentValue from Current→LastUsed only; enabled-gated thought current; omission arms | closed |
| T-06-12 | Tampering | acpport/agent | low | mitigate | SDK union variant only; Validate asserted | closed |
| T-06-13 | Tampering/honesty | acpfront/auth | high | mitigate | watcher BEFORE push; verdict = registry read-back Await, ack never consulted; read-back-negative arm. NOTE: the round-3 eager mint (auth.go:266-275) supersedes 06-05 PLAN's mint-ordering prohibition wording — deviation documented in-code with its root cause (create+close alone never materializes the account provider) and field-proven by the passing real run; the declared mitigation (registry-is-the-verdict) is intact | closed |
| T-06-14 | Denial of Service | interact/keyringprobe | medium | mitigate | 5 s probe timeout; LookPath gate naming libsecret-tools; single argv source; scrubbed env; never-hangs vector | closed |
| T-06-15 | Information Disclosure | interact/familysel | high | mitigate | presence booleans + class codes only; token value never surfaced | closed |
| T-06-16 | Tampering/honesty | acpfront/auth | medium | mitigate | coded "log in via the ZCode desktop first" before any traffic; no-login arm test-locked | closed |
| T-06-17 | Tampering | acpfront/front | medium | mitigate | validate-first set_config_option (session/configId/value); backend call only after validation | closed |
| T-06-18 | DoS (turn failure) | acpfront/front | medium | mitigate | level carry on model switch; thought switch on setModel path; persist:false; goldens byte-lock | closed |
| T-06-19 | Tampering/honesty | acpfront/front | medium | mitigate | store overwritten from the mutation reply BEFORE the rebuild (TestSetConfigOptionReEmit) | closed |
| T-06-20 | DoS (turn death) | interact/responder | high | mitigate | keySource error routes to the coded ERROR arm, never headersApplied:false; PAT-less arm answers coded error | closed |
| T-06-21 | Information Disclosure | interact/responder | high | mitigate | accountScope joins only when non-empty (api-key answer byte-identical); synthetic PAT shapes in goldens; scan-secrets green | closed |
| T-06-22 | EoP (stale PAT serving) | hostsvc/headers_pat | high | mitigate | scope gate FIRST (ErrPatScopeInvalidated before any token); re-checked against the fresh mint | closed |
| T-06-23 | DoS (re-mint loop) | hostsvc/headers_pat | medium | mitigate | exactly one Invalidate+Ensure per ask; ErrPatRejectionLoop stops mints at 2 | closed |
| T-06-24 | Information Disclosure | test/realdesktop | high | mitigate | scrub-capture.mjs gate before commit; scrubbed capture committed; scan-secrets green on tree | closed |
| T-06-25 | DoS (spend) | experiments ledger | medium | mitigate | ledger row with planned D-13 cap + full gap-cycle actuals (attempts 4-8 mint re-runs, zero model tokens) | closed |
| T-06-26 | Tampering/honesty | test/realdesktop | high | mitigate | assertions read registry/live state, never acks; failed attempts recorded honestly with root causes | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-06-01 | T-06-07 | Duplicate `zcode-api-key` mint race is process-local (the single-flight Ensure serializes within one wrapper process), cosmetic, and self-heals via the list-first search on the next mint; the desktop shows the same behavior. PLAN 06-02 rationale. | orchestrator (auto, ASVS L1 low) | 2026-11-22 |

---

## Surface Inventory (post-plan additions)

- `interaction/requestOfficialMcpAuthHeaders` honest-decline policy row (interact/responder.go — structured `{ok:false, reason:"official_auth_unavailable"}` RESULT + the `official-mcp-auth-decline` corpus golden). Added in-phase during the real run (EXP-CODINGPLAN-CYCLE-01 attempts 1-2 root cause: -32601 stalled runtime materialization). Attack surface: conservative decline-only — never fabricates success, never echoes crypto error text; the desktop's own host uses the same shape for its untrusted/unavailable arms. No STRIDE row existed at plan time; audited here as part of the register review (no finding).

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-22 | 27 | 26 | 1 (low, accepted → AR-06-01) | gsd-security-auditor (ASVS L1, mitigate-verification mode; every mitigation traced to code, every test lock run live under -race) |

**Notes:** the real-run evidence (EXP-CODINGPLAN-CYCLE-01 attempt 8 PASSED) was audited as part of T-06-24/25/26. The round-3 eager-mint ordering deviation from the 06-05 PLAN wording is recorded at T-06-13. Review dispositions: WR-02/03/06 resolved in-phase (verified), WR-01/04/05 + IN-01..04 routed open to Phase 7 (advisory, no register impact).
