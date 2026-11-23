# Phase 4 — API Coverage Declaration

**Phase:** 04-acp-front-minimal-chat-loop
**Declared:** 2026-11-20
**Kind:** reasoned declaration (no matrix — see rationale)

## Declaration

Phase 4 does NOT integrate an external service API, so no API-coverage matrix is
produced. The honest reading of the detector hit is:

`coder/acp-go-sdk` v0.20.0 is the **ACP protocol implementation itself** (framing,
typed schema surface, dispatch, cancel bookkeeping) — the same category as the
Phase-3 hand-rolled JSONL codec for the back leg, not an HTTP/API client over a
remote service. The phase's only dependency addition is this protocol SDK
(D-01 audit PASSED, recorded in 04-RESEARCH.md "D-01 Verdict").

What Phase 4 actually "integrates":

| Surface | Kind | Coverage mechanism |
|---------|------|--------------------|
| `acp.NewAgentSideConnection` / `acp.Agent` (12 methods) | Protocol SDK, agent side | Every handler exercised by an SDK `ClientSideConnection` round-trip over pipes (fake-driven, in-process — RESEARCH Pattern 6); contract locks per CORE-01..05 in plans 04-02..04-06 |
| `acp.NewClientSideConnection` / `acp.Client` | Protocol SDK, client side (test counterparty) | Used as the honest client in all front-tier tests; keeps the counterparty spec-true |
| ZCode app-server (back leg) | Local process, scripted | The Phase-3 fake app-server corpus — no real service is touched in CI (STATE.md standing constraint: CI never calls api.z.ai) |

The z.ai HTTP surface (api.z.ai PAT chain) is Phase 6 scope (AUTH-02) and will get
its own coverage treatment then.

## Edge/prohibition probe fallback

The specless probe fallback ran for this phase (no SPEC sections supplied). The 10
edge-probe items and the prohibition recall/precision results are authored into the
PLAN.md `must_haves` blocks (truths, one backstop marker, descriptor-less
prohibitions) with the equality ledger per plan:

- CORE-01: adjacency / empty / ordering resolved explicit (plan 04-03); concurrency resolved backstop (plan 04-03)
- CORE-03: adjacency / empty / ordering resolved explicit (plan 04-04)
- CORE-02, CORE-04, CORE-05: `unclassified` — stay unresolved, surfaced as flagged assumptions (plans 04-03, 04-04, 04-05 respectively)
- Prohibitions kept (3, descriptor-less, flagged-unverified): advertise=implemented (04-03), never synthesize chunks (04-04), authenticate never fakes success (04-03)

Coverage: 10 applicable → 7 resolved (6 explicit + 1 backstop), 3 unresolved flagged.
Equality: 10 = 7 authored + 3 flagged. No silent drops.
