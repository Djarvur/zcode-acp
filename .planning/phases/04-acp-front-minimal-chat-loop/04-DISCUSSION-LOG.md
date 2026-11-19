# Phase 4: ACP Front — Minimal Chat Loop - Discussion Log

> **Audit trail only.** Decisions captured in CONTEXT.md.

**Date:** 2026-11-20
**Phase:** 4-ACP Front — Minimal Chat Loop
**Mode:** `--auto` (standing directive)

---

## ACP leg implementation base

| Option | Description | Selected |
|--------|-------------|----------|
| coder/acp-go-sdk v0.20.0 behind internal/acpport, gated by Package Legitimacy Audit; hand-roll pre-approved fallback | STACK HIGH-confidence recommendation + seam containment vs supply-chain weight of v0.x dep | ✓ (audit decides at planning) |
| Hand-roll unconditionally | Zero deps but reimplements a large typed spec surface | fallback |
| Community SDKs (ironpark/BrokkAi) | Less proven, no official org backing | ruled out |

**User's choice:** *(auto)* Audit-gated SDK-first

## Capability honesty / session-new / streaming / cancel

| Decision | Choice |
|----------|--------|
| Capabilities | Exactly what exists: protocolVersion 1, both authMethods advertised (functional Phases 5–6), no fs/terminal, no slash commands |
| session/new | create + setModel follow-up (wire-asserted), mcpServers accept-and-ignore |
| Streaming | live deltas (never buffered-flush), stopReason after flush, thought only on backend events |
| Cancel | bypass stop, exactly-once cancelled, second-prompt rejected |

## Claude's Discretion
- acpport API shape, test structuring, agentInfo details, wave decomposition

## Deferred Ideas
- permission bridging (P5), elicitation (P5+), configOptions (P6), subscribe watcher (slot reserved)
