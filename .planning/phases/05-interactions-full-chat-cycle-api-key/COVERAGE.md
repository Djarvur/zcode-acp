# Phase 5 — API Coverage Declaration & Probe Ledger

**Phase:** 05-interactions-full-chat-cycle-api-key
**Declared:** 2026-11-20
**Kind:** reasoned declaration (no matrix — see rationale) + specless probe ledger (finalized)

## API Coverage Declaration

Phase 5 does NOT integrate a NEW external service API, so no API-coverage matrix is
produced. The detector fires on wording ("chosen outcome", request/response
vocabulary), but the honest reading is:

- The wire ops of this phase (`provider/updateAccountConfig`,
  `provider/refreshRegistry`, `interaction/*`, `session/resume|close`) travel over
  the EXISTING child transport — the Phase-3 JSONL codec Conn to the local
  app-server process. No new external API client is created.
- The z.ai HTTP surface (api.z.ai PAT chain) is Phase-6 scope (AUTH-02) and gets its
  own coverage treatment then.
- The `secret-tool` subprocess (D-06 CredentialDecrypt seam) is a local desktop
  integration (libsecret CLI over the login keyring), used exclusively by the
  guarded real tier — not a remote API client, never CI.

| Surface | Kind | Coverage mechanism |
|---------|------|--------------------|
| ZCode app-server child (interaction/*, snapshot push, refreshRegistry, resume/close) | Local process wire, Phase-3 codec | Fake app-server corpus rows with `expect_answer` locks per plan (05-01/02/04/05/06/08); the composed full-cycle E2E (05-08); guarded real tier (05-09) |
| acp-go-sdk `RequestPermission` / tool-call helpers / Resume/Close | Protocol SDK (already-pinned v0.20.0) | Composed harness round-trips against a real SDK `ClientSideConnection` (05-02/05-04); SDK-import boundary test stays the seam lock |
| `secret-tool` + stdlib AES-256-GCM (D-06) | Local keyring subprocess, real tier only | Synthetic-vector unit locks + hygiene locks (05-07); live exercise only in the guarded run (05-09) |
| z.ai HTTP (api.z.ai PAT chain) | NOT IN THIS PHASE | Phase 6 (AUTH-02) |

## Assumption-Delta Advisory

`assumption-delta scan "5"` returned `detected` on one signal — the term "chosen" in
success criterion 3 ("the chosen outcome is forwarded"). Reading: this is design
vocabulary (the client's selected permission outcome), not an unresolved
alternatives assumption. No action taken; recorded here for the audit trail.

## Specless Probe Fallback — Edge Ledger (finalized)

8 applicable edge-probe items; **all 8 resolved explicit** in plan `must_haves`
truths with named tests. Equality: 8 = 8 authored + 0 flagged. No silent drops.

| # | Requirement | Probe (category) | Resolution | Plan/Task |
|---|-------------|------------------|------------|-----------|
| 1 | CORE-06 | adjacency — "do equal/touching merge, collide, or separate?" | Equal `(method, params.requestId)` merges into ONE dedupe entry answered once to the registering wire id; distinct requestIds never share an entry | 05-01 T2 (staged asserts + reannounce-dedupe golden) |
| 2 | CORE-06 | empty — result for empty/null input | Params without requestId → NO dedupe entry, answered per envelope (safe: re-announces always carry requestId) | 05-01 T2 (staged assert) |
| 3 | CORE-06 | ordering — specified and stable when elements compare equal? | Fan-out delivery order is registration order (A then B); per-key entries answered exactly once in claim order | 05-03 T1 (TestRouterFanout) + 05-01 dedupe race |
| 4 | CORE-06 | concurrency — interrupted/parallel guarantee | Concurrent same-key claims run the handler exactly once, one reply; bounded map evicts on answer/close/TTL; conn close owes nothing on a dead stream | 05-01 T2 (TestResponderDedupeRace, -race) |
| 5 | AUTH-01 | concurrency | Interrupted/cancelled authenticate → coded error, NO recorded success; re-authenticate is a clean retry (no half-authenticated state observable) | 05-07 T2 arm (e) |
| 6 | TOOL-01 | concurrency | Queue-ceiling breach mid-stream fails the turn LOUD (coded cause); same-toolCallId updates coalesce keep-latest; text/thought lossless until the loud ceiling | 05-04 T3 (TestStreamBackpressure three legs) |
| 7 | TOOL-02 | concurrency | Cancel racing the client decision yields exactly ONE outcome (selected response or -32021; never both, never neither) — reply-once under -race | 05-02 T3 (race probe) |
| 8 | SESS-01 | concurrency | Close while a turn is active cancels first (bounded wait); close racing a second prompt yields exactly one clean outcome; close-vs-resume guarded by the registry (unregistered ids never reach the wire) | 05-05 T2 (close-race + rejected-prompt probes) |

## Prohibition Recall/Precision (descriptor-less, flagged-unverified)

Kept — 7 bespoke prohibitions, authored into the owning plans' `must_haves.prohibitions`:

1. No auto-allow: outcomes the backend options do not define (unknown/missing optionId, malformed, expiry) resolve deny; the only allow path is the client's explicit selection (05-02)
2. Never synthesize tool/permission frames — every client frame traces to one received backend envelope (05-02, 05-04)
3. Auth success only on registry read-back, never the ack (05-06, 05-07)
4. The failure-shaped success for runtime headers (the headersApplied:false arm) is never emitted — failures are coded errors (05-06)
5. Unknown interaction/* answered protocol-error (-32601 naming the method) — never fabricated success, never a crash (05-01)
6. CI tiers never touch the keyring or the account; keyring/key material stays out of CI paths (05-07, 05-09)
7. Resume never replays/synthesizes pre-resume history (05-05)

Dropped with breadcrumbs (canon → existing tooling/threat register; routine → edge probe/code review):

- Credential material in logs/captures — canon V7 secrets-logging → `/gsd:secure-phase` + mitigated as T-5-03/T-5-06/T-5-15/T-5-17/T-5-21 in the plan threat models
- Re-announce flood DoS — canon DoS → mitigated as T-5-01 (bounded dedupe map)
- Key visible on argv / in child env — canon V2 → mitigated in 05-07's hygiene contract (T-5-17)
- Close-failure reporting semantics — routine engineering (best-effort + structural log is the researched design)
- Subagent/batch tool-card misattribution — routine (explicit mapping spec, 05-04)
- Deadlines silently disabled — routine (explicit per-method deadline table is the CORE-06 contract itself)

Equality: prohibitions 7 kept + 6 dropped-with-breadcrumbs = 12 surfaced; 0 silent drops.

## Flagged Assumptions (carried, never auto-dismissed)

| # | Assumption | Where it lives | Disposition |
|---|------------|----------------|-------------|
| A1 | requestProviderRuntimeHeaders never fires on zai-api personal turns on 3.18.2 (3.17.1 runner gate + absence in all api-key captures) | 05-06 (probe ledger) | Responder answers correctly regardless (CORE-06); the corpus row locks the shape; the real run's capture confirms |
| A2 | secret-tool exists on target v1 machines (verified THIS machine only) | 05-07, 05-09 (probe ledgers) | Loader failure degrades honestly per D-06 ({ok:false} → honest turn failure); godbus vault is the Phase-6 upgrade |
| A3 | Tool names outside the mapping table render as kind `other` (cosmetic) | 05-04 (probe ledger) | Mapping table is data; correctable without architecture change |
