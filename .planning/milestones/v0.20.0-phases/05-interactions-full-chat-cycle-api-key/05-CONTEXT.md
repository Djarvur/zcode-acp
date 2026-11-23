# Phase 5: Interactions & Full Chat Cycle (api-key) - Context

**Gathered:** 2026-11-20
**Status:** Ready for planning
**Mode:** `--auto` — decisions auto-selected (recommended defaults); see `05-DISCUSSION-LOG.md`

<domain>
## Phase Boundary

Phase 5 delivers the **host-managed interaction responder always-on** (CORE-06: every
`interaction/*` answered promptly — dedupe re-announcements, per-method deadlines,
unknown methods get protocol errors, never crash), **tool-call streaming** (TOOL-01:
`tool_call`/`tool_call_update` with title, kind, status transitions, file diffs for edit
tools), **permission bridging** (TOOL-02: backend `interaction/requestPermission` →
client `session/request_permission` with allow/reject options; chosen outcome forwarded;
ACP-side cancel resolves pending as `cancelled`), **api-key auth path** (AUTH-01:
`provider/updateAccountConfig` + `interaction/requestProviderRuntimeHeaders` answered
correctly, registry state read back — never trust the ack), and **session lifecycle**
(SESS-01: `session/resume` reattach without replay + `session/close` child cleanup).
Full cycle new → prompt with tools + permission decision → resume → close proven on a
real api-key account (success criterion 5).

Out of scope: OAuth coding-plan chain (Phase 6), configOptions model picker (Phase 6),
session/delete & fork compat no-ops (Phase 7), real-client UI E2E (Phase 7 QA-03).

Also inherits Phase 4 review-routed work (04-REVIEW-DISPOSITION.md, all OPEN):
WR-01 OnRouterEvent per-session wiring, WR-02 atomic sink publish, WR-03 streaming
backpressure design, WR-04/07 session teardown (SESS-01 lands WITH teardown —
NewMachine observer remove), WR-05/06 wording + dropped-block logging, IN-01..05.

</domain>

<decisions>
## Implementation Decisions

### Interaction Responder Architecture
- **D-01:** The responder is a **policy object** per ARCHITECTURE (Pattern 3): every
  server→host method gets a handler chosen by auth mode; **per-method deadlines**
  (180 s for requestProviderRuntimeHeaders per Phase-1-verified transport facts; 15 s
  requestRuntimePreferences already landed); **dedupe 1 s re-announcements by
  requestId** (Phase-1-verified broker behavior); unknown methods answered with a
  protocol error (-32601-class) — never crash (CORE-06). Phase-3's
  requestRuntimePreferences static answer migrates into this policy object (same
  behavior, new home).

### api-key Auth Path
- **D-02:** AUTH-01 answers `provider/updateAccountConfig` with the api-key snapshot
  (exact-revision builtin per install-discovery.md formula) and
  `interaction/requestProviderRuntimeHeaders` with `{headersApplied:true,
  requestAuth:{apiKey, apiKeyId}}` — the PAT comes from the account config the user
  supplies via the new `authenticate{zai-api-key}` front method. **Verification = read
  `provider/refreshRegistry`/registry state back**, never the `received` ack (PITFALL 1).
  Requires ONE real account round-trip to prove end-to-end (success criterion 4) —
  quota-minimal (registry read-back + one tiny tool-less turn), the Phase-1 UAT AR
  precedent covers the spend format.

### Tools & Permissions
- **D-03:** Tool lifecycle events map to `tool_call` (start) / `tool_call_update`
  (progress/completion) with title/kind/status from the dialect's event taxonomy;
  **file diffs for edit tools** rendered as ACP diffs (TOOL-01). Permission bridging
  maps `interaction/requestPermission` options to ACP
  `session/request_permission` option kinds; the client's decision forwards to the
  backend; **ACP-side cancel resolves the pending request as `cancelled`** and the
  backend turn ends (TOOL-02). Real-tool verification: one tiny real turn exercising a
  file-read tool on the api-key account (quota-minimal, same AR precedent).

### Session Lifecycle
- **D-04:** SESS-01 lands WITH the Phase-4-review teardown work: `session/resume`
  reattaches (backend `session/resume` — no replay), `session/close` runs child-side
  cleanup + releases registry/machine/observer (04-REVIEW WR-04/WR-07: NewMachine
  exposes observer remove). `session/load`/fork stay deferred (v2/Phase 7).
  Phase-4 review items land here by scope: WR-01 (per-session OnRouterEvent via
  fan-out), WR-02 (atomic sink publish), WR-05/06 (wording + dropped-block logging).

### Credential-Cipher Real-Tier Strategy (research P0)
- **D-06:** A real `session/send` fires `interaction/requestCredentialCipher` mid-turn; tracked-no-answer blocks success criteria 4/5. Adopt the research recommendation: **CredentialDecrypt interface seam answered via `secret-tool lookup` + stdlib AES-256-GCM** (the field-verified predecessor-shim path; the backend asks, we decrypt with the user's own login-keyring key, we answer — exactly what the desktop itself does; godbus CredentialVault remains the Phase 6 transport upgrade). Keyring-locked/headless degrades to tracked-no-answer + honest turn failure (documented, never fabricated). CI tiers never touch the keyring.

### Real-Account Discipline
- **D-05:** Real api-key operations follow the Phase-1 AR precedent: minimal turns,
  ledgered in experiments/README quota section, **CI never** runs them (guarded manual
  tier); credentials from local config (never committed); every real-account claim
  gets a scrubbed capture + V-EXP id (evidence discipline unchanged).

### Claude's Discretion- requestPermission host deadline: 300 s configurable, expiry → deny (OQ2)
- WR-03 backpressure: bounded per-turn queue + fail-loud memory ceiling (OQ3)
- authenticate without configured key → honest error naming ZCODE_ACP_API_KEY (OQ4)
- dead registry fields CancelTurn/Subscribed → delete (OQ5)

- Policy-object API shape; tool-event mapping details within the dialect taxonomy;
  test design (fake-driven permission/tool scenarios + the real-account smoke tier)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning & Research (in repo)
- `.planning/REQUIREMENTS.md` — CORE-06, AUTH-01, TOOL-01, TOOL-02, SESS-01
- `.planning/ROADMAP.md` — Phase 5 success criteria (5 items)
- `docs/protocol/app-server-dialect.md` — interaction/* surface (7 methods incl.
  requestPermission/requestRuntimeHeaders/requestUserInput), tool lifecycle events,
  permission shapes, error codes, serial queue + stop bypass
- `.planning/research/ARCHITECTURE.md` — Pattern 3 interaction responder (policy
  object, deadlines, dedupe), tool streaming notes
- `.planning/research/PITFALLS.md` — pitfalls 1 (ack≠acceptance), 8 (cancel races,
  180s credential queue)
- `.planning/phases/04-acp-front-minimal-chat-loop/04-REVIEW-DISPOSITION.md` — the 12
  OPEN findings Phase 5 owns (WR-01..07, IN-01..05)
- `.planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-VERIFICATION.md`
  — the interaction machinery to extend
- agentclientprotocol.com v1 spec — session/request_permission, tool-calls pages

### External Ground Truth (outside the repo)
- ZCode 3.17.x clone (interaction broker: 1 s re-announce, deadline behavior; provider
  registry service) — cite-and-point only
- FINDINGS.md + field-notes oauth-coding-plan-shim.md (api-key path field-verified
  shapes: requestProviderRuntimeHeaders reply, updateAccountConfig)
- Real api-key account: credentials in local config (never committed)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- internal/zcode router (per-method deadlines landed in 03-02; tracked-no-answer for
  credential-cipher) — the responder extends this into the policy object
- internal/turn machine (exactly-once, bypass cancel) — tool/permission events feed it
- internal/acpfront harness — permission/tool scenarios extend the corpus
- internal/supervisor + fake binary — the real-account smoke tier composes them

### Established Patterns
- TDD RED→GREEN machine evidence; exactly-once semantics test-locked
- Provenance: real-account claims get scrubbed captures + V-EXP ids
- Zero-dep discipline (godbus deferred to Phase 6; nothing new needed in Phase 5)

### Integration Points
- acpfront gains: authenticate{zai-api-key}, session/request_permission (client→agent
  side handled), tool_call/update emission into session/update stream
- zcode router gains: requestPermission + requestProviderRuntimeHeaders handlers
- supervisor/real-smoke: api-key config before turn

</code_context>

<specifics>
## Specific Ideas

- The interaction responder is a correctness PRECONDITION (headersApplied:false =
  guaranteed turn failure) — its fake-scenario tests precede the real-account tier.
- requestPermission deadline expiry answers deny/cancel to the backend (never hang).
- Permission decision forwarding: the client's allow/reject choice maps back verbatim;
  an ACP cancel during the pending window resolves backend-side as cancelled.
- Registry read-back helper (`awaitRegistryContains`) per PITFALL 1 — reused by Phase 6.

</specifics>

<deferred>
## Deferred Ideas

- OAuth chain + keyring vault (Phase 6); configOptions/model picker (Phase 6)
- elicitation/create preference (Phase 5+ if client capability-gated — needs a real
  client to test; defer to Phase 7 integration)
- session/load/fork (v2)

</deferred>

---

*Phase: 5-Interactions & Full Chat Cycle (api-key)*
*Context gathered: 2026-11-20*
