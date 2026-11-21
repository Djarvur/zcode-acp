---
phase: "5"
slug: "interactions-full-chat-cycle-api-key"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-22"
---

# Phase 5 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Real z.ai account ↔ wrapper (api-key) | authenticate{zai-api-key}: push/refresh/probe-create read-back chain over the desktop's app-server; ack never trusted | api key (env-sourced in manual runs; registry-state verdict only) |
| Login keyring ↔ cipher decrypt | secret-tool lookup (sha256-of-resolved-path username) + AES-256-GCM over the desktop's enc:v2:os blobs | keyring key + credential plaintext (in-process only) |
| Raw captures ↔ repository | scrub-capture.mjs is the only path into experiments/captures/; scan-secrets.sh gates every commit; raw captures born outside | scrubbed JSONL (live key never enters the repo) |
| Desktop agent ↔ interaction responder | host-managed responder always on: permission/cipher/headers/mcp asks answered per contract | permission decisions, cipher values, header material |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-5-SC | Tampering | dependency pin | high | mitigate | go.mod single require acp-go-sdk v0.20.0; mise tidy-diff gate | closed |
| T-5-01 | DoS | interact dedupe | medium | mitigate | bounded table + TTL sweep + tombstone (dedupe.go:64-105); golden row | closed |
| T-5-02 | Tampering/DoS | unknown host methods | high | mitigate | CodeMethodNotFound handled reply (responder.go:211-215); golden | closed |
| T-5-03 | Info disclosure | interact logging | medium | mitigate | structural-only log fields (responder.go, permission.go) | closed |
| T-5-04 | EoP | permission options | high | mitigate | deny-by-default + byte-verbatim passthrough + -32021 (permission.go:88-127) | closed |
| T-5-05 | DoS | permission pendings | medium | mitigate | session-scoped table + fan-in on all three arms (front.go:433,479; lifecycle.go); close/cancel goldens | closed |
| T-5-06 | Info disclosure | acpfront logging | low | mitigate | counts/kinds/ids/errs only | closed |
| T-5-07 | DoS | router observers | medium | mitigate | ordered fan-out (conn.go:153); TestRouterFanout | closed |
| T-5-08 | DoS | machine observers | medium | mitigate | CloseObservers idempotent (machine.go:209); baseline test | closed |
| T-5-09 | DoS | turn stream | high | mitigate | turnPump 64 MiB ceiling + breach cancel (turnctx.go); TestStreamBackpressure | closed |
| T-5-10 | Info disclosure | chunk drops | low | mitigate | kind-only warn at drop site (stream.go) | closed |
| T-5-11 | Tampering | tool paths | low | mitigate | resolveAbs joins session cwd (stream.go:367-375) | closed |
| T-5-12 | DoS | orphan sessions | medium | mitigate | WR-04 best-effort orphan close (front.go:252-279); newsession-orphan-close.golden | closed |
| T-5-13 | Spoofing | unknown sessions | low | mitigate | registry.Get coded rejection pre-backend (front.go:326,419) | closed |
| T-5-14 | Tampering | auth verdict | high | mitigate | readback.go Watch/OfferCreateReply/Await; never ack-derived (:97); TestAuthReadback | closed |
| T-5-15 | Info disclosure | snapshot key | high | mitigate | key in-memory only (snapshot.go:245-256); scan-secrets | closed |
| T-5-16 | Tampering | builtin revision | medium | mitigate | BuiltinRevision reuse, never re-derived (snapshot.go:133-135) | closed |
| T-5-17 | Info disclosure | keyring loader | critical | mitigate | argv carries hash only; scrubbed env; drained stdout; never logged (cipherdecrypt.go:341-393); hygiene tests | closed |
| T-5-18 | Spoofing | authenticate | high | mitigate | push→refresh→probe→Await chain; errAPIKeyUnconfigured (auth.go:150-173); 5-arm test | closed |
| T-5-19 | DoS | cipher loader failure | medium | mitigate | honest {ok:false} decline arm; loader-failure test | closed |
| T-5-20 | Tampering | golden authority | medium | mitigate | full-cycle + per-leg goldens authoritative; TestFullCycle | closed |
| T-5-21 | Info disclosure | real-cycle key | critical | mitigate | env-sourced key, never embedded/logged (apicycle_test.go:1-30); scrub + scan gates | closed |
| T-5-22 | DoS | quota guard | medium | mitigate | realdesktop tag + REAL_E2E double guard; honest skip | closed |
| T-5-23 | Tampering | nil replies | low | mitigate | structural skip (conn.go:501-502); TestReplyStructuralSilence | closed |
| T-5-24 | Info disclosure | resume leak | medium | mitigate | isClosing gate + re-check + IsAbs cwd (lifecycle.go:72,81,126); 3 tests | closed |
| T-5-25 | DoS | stale fan-in | medium | mitigate | pump-identity re-check under slot lock (front.go:453-485); TestCancelFanInStaleArmSkips | closed |
| T-5-26 | Info disclosure | diagnostic capture | critical | mitigate | double guard + raw born outside; scrubbed DIAG-01 committed; scan 0 | closed |
| T-5-27 | Tampering | push payload | high | mitigate | branch-(c) verdict ledgered; providers stay empty — no key in push (auth.go:180-189) | closed |
| T-5-28 | DoS | diagnostic spend | low | mitigate | zero session/send in the diagnostic file (structural) | closed |
| T-5-29 | Info disclosure | committed captures | critical | mitigate | CYCLE-01/02/03 committed scrubbed; scan-secrets exit 0 | closed |
| T-5-30 | DoS | run quota | medium | mitigate | ledger actuals = D-05 minimum; canonical command; double guard (README.md:195-230) | closed |
| T-5-31 | Repudiation | failed-run evidence | medium | mitigate | attempts 1-2 failures recorded with scrubbed evidence + ledger lines | closed |
| T-5-32 | Repudiation | retry discipline | medium | mitigate | errors.As Code==InvalidRequest only; else fail loud (apicycle_test.go:943) | closed |
| T-5-33 | Info disclosure | ask tap | high | mitigate | method + UTC only, never params (apicycle_test.go:740-742) | closed |
| T-5-34 | Info disclosure | monotonic captures | critical | mitigate | -03 NEW file; -01/-02 byte-intact; twice-scan discipline | closed |
| T-5-35 | DoS | run count | medium | mitigate | exactly-one-run-per-checkpoint; one run's actuals ledgered | closed |
| T-5-36 | Repudiation | capture pointers | medium | mitigate | earlier pointers never repointed; ledger names -03 explicitly | closed |
| T-5-37 | Tampering | fan-in scope | medium | mitigate | session-scoped buckets + atomic take under turnMu (permission.go:38,149-177; front.go:433,479); TestCancelFanInSkipAndScope | closed |
| T-5-38 | DoS | lock deadlock | low | mitigate | one-way turnMu→bridge.mu documented; bridge never takes turnMu; -race green | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

No accepted risks.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|--------------|--------|------|--------|
| 2026-11-22 | 39 | 39 | 0 | gsd-security-auditor (L1 grep-depth; -race full suite green; scan-secrets exit 0) |
