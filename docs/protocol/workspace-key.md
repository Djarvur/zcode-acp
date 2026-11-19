# workspaceKey Semantics

**Domain:** ZCode app-server `session/create` workspace identity — wire schema, derivation convention, experiment record
**Researched:** 2026-11-19
**Confidence:** HIGH — the source convention was read verbatim from the 3.17.1 clone and confirmed on the wire by experiment EXP-WORKSPACEKEY-01 against the shipped 3.18.2 binary

Provenance legend and tag semantics are defined once in [README.md](README.md);
this doc references that legend instead of redefining it. Ground-truth hierarchy
per D-01: the shipped 3.18.2 desktop bundle is V-SRC ground truth; the 3.17.x
public-source tree (this doc cites the **3.17.1** clone) is corroborating only.

## Wire schema

- `workspaceKey` is a **non-empty string** and is **required** in the
  session-create workspace shape that carries it alongside `workspacePath`
  [V-SRC packages/shared/src/zcode-protocol/index.ts#L2762 (3.17.1)].
- A second workspace-ref shape declares it **optional**
  (`workspaceKey: nonEmptyString.optional()`)
  [V-SRC packages/shared/src/zcode-protocol/index.ts#L2784 (3.17.1)] —
  i.e. the field is host-supplied metadata, not something the server derives.
- **Empty/blank keys are rejected by the schema type itself**: `nonEmptyString`
  rejects the empty string, so a blank `workspaceKey` is not wire-acceptable
  where the required shape applies [V-SRC packages/shared/src/zcode-protocol/index.ts#L2762 (3.17.1)].
- The server treats the key as an **opaque host-chosen string**: no validation
  against the path, no transformation on the wire — the create reply echoes the
  workspace ref verbatim [V-EXP EXP-WORKSPACEKEY-01].

## Derivation convention

The client entrypoint derives the key with a fixed convention, quoted verbatim
from the 3.17.1 source (original comment preserved; translation below)
[V-SRC apps/zcode-cli/packages/bootstrap/src/zcode-protocol-entrypoint.ts#L219-L236 (3.17.1)]:

```ts
const workspaceIdentity = (options.env ?? process.env)[ZCODE_WORKSPACE_IDENTITY_ENV]?.trim();
...
// workspaceKey 必须遵守仓库约定 `workspaceIdentity?.trim() || workspacePath`，
// 否则同路径不同 identity 的远端 workspace 在审计上下文里无法区分。
// 注意：agent 进程当前没有 identity 来源，因此实际多为 undefined，key 退化为 path；
resolveWorkspace: ({ workspaceIdentity, workspacePath }) => {
  const path = workspacePath ?? options.cwd;
  if (!path) return undefined;
  const identity = workspaceIdentity?.trim();
  return { ...(identity ? { workspaceIdentity: identity } : {}),
           workspaceKey: identity || path,
           workspacePath: path };
},
```

Translation of the comment: "workspaceKey must follow the repo convention
`workspaceIdentity?.trim() || workspacePath`, otherwise remote workspaces with
different identities on the same path cannot be distinguished in audit context;
note that agent processes currently have no identity source, so in practice it
is mostly undefined and the key degrades to the path."

- **Environment override:** the identity comes from the env var
  `ZCODE_WORKSPACE_IDENTITY` (host-injected at spawn; used for isolation/audit,
  not for file execution)
  [V-SRC packages/shared/src/official-mcp-auth.ts#L149 (3.17.1)].
- **Encoding/whitespace semantics:** the identity term is **trimmed before
  use** (`workspaceIdentity?.trim()`); a whitespace-only value trims to the
  empty string and falls through `||` to the path. The key itself is a plain
  string — no encoding is applied or required.
- **Degradation:** agent processes have no identity source today, so the
  effective convention in practice is `workspaceKey = workspacePath`.

## Experiment record

Experiment **EXP-WORKSPACEKEY-01** — capture
`experiments/captures/EXP-WORKSPACEKEY-01.jsonl` (scrubbed through the
canonical scrubber; raw capture born and kept outside the repository).

- **Setup:** app-server spawned over stdio from the shipped 3.18.2 desktop
  bundle (`ELECTRON_RUN_AS_NODE=1 <bundle>/zcode <bundle>/resources/glm/zcode.cjs
  app-server --stdio`); the bundle copy probed is **pristine** — byte-identical
  (`cmp`) to its unmodified backup at experiment time (D-01 provenance fact,
  reused by DISC-03). The capture header pins `desktopVersion` 3.18.2 and
  `builtinRevision` 44 read from the active builtin config at capture time
  (stale-evidence guard). Workspace: a scratch directory outside the repository
  (`/tmp/…`). Registry-level only — **no turn, no quota spent**.
- **Verdict 1 — path-as-key accepted:** `session/create` with
  `workspace{workspacePath: P, workspaceKey: P}` (P = the scratch directory)
  returned `result.session.sessionId` and the session reached `ready`
  [V-EXP EXP-WORKSPACEKEY-01].
- **Verdict 2 — arbitrary host-chosen key accepted:** a second create with the
  **same** `workspacePath` and `workspaceKey: "probe-arbitrary-host-key-01"`
  also returned `result.session.sessionId` and reached `ready`; both sessions
  coexisted [V-EXP EXP-WORKSPACEKEY-01]. Together: **host-opaque semantics** —
  the server neither validates the key against the path nor derives one.
- **Determinism/idempotency:** the convention is a pure function of
  (identity, path) — the same input yields the same key; on the wire the reply
  echoes the workspace ref verbatim, so what the host sends is what the server
  stores and returns [V-SRC apps/zcode-cli/packages/bootstrap/src/zcode-protocol-entrypoint.ts#L219-L236 (3.17.1)]
  [V-EXP EXP-WORKSPACEKEY-01].
- **Host-obligation discovery (first run):** with no host responder, both
  creates failed `-32603 "ZCode Protocol client connection closed"` — on 3.18.2
  `session/create` issues the server→host **request**
  `session/requestRuntimePreferences` (scope `runtime-materialization`; 15 s
  protocol deadline
  [V-SRC packages/shared/src/zcode-protocol/index.ts#ZCODE_SESSION_RUNTIME_PREFERENCES_REQUEST_TIMEOUT_MS (3.17.1)])
  and the host must answer it (minimal accepted answer:
  `{nativeSearchEnhancementsEnabled:false, memoryEnabled:false}`, per the 3.17.1
  reference host fixture
  [V-SRC packages/desktop/scripts/mcp-apps-host-fixture/agent.ts#L82-L86 (3.17.1)]).
  The same capture also records previously-unsurveyed server→host surface for
  DISC-01: repeated `interaction/requestOfficialMcpAuthHeaders` requests and
  `process/mcpTelemetry` notifications during create [V-EXP EXP-WORKSPACEKEY-01].
- **Shared-state caveat:** the experiment shared `~/.zcode` state with the live
  desktop (credentials, runtime provider tree). Sessions were left idle, nothing
  was mutated, and the capture header pins date + revision so any desktop
  auto-update mid-phase stays detectable; treat registry/session state as
  read-and-record, not assert-and-hold.

## Host-freedom note

In host-managed mode the ACP wrapper **is the host**: it spawns the app-server
and chooses the `workspaceKey` freely — the server accepts any non-empty string
[V-EXP EXP-WORKSPACEKEY-01]. The documented convention
(`workspaceIdentity?.trim() || workspacePath`) is the **compatible choice**, not
a wire requirement: adopting it costs nothing, keeps same-path
different-identity workspaces distinguishable in audit context, and matches the
behavior of the reference host implementation. A wrapper that simply passes the
workspace path as the key reproduces the default degradation exactly.

## Sources & provenance

Legend: see [README.md](README.md). All 3.17.x citations below were read from
the public-source tree at version **3.17.1** (package.json) and are corroborating
evidence for shipped-binary behavior; the shipped 3.18.2 bundle is ground truth
(D-01) and was verified pristine at experiment time (byte-compare recorded in
the Experiment record).

| Id | Origin |
|----|--------|
| `[V-SRC packages/shared/src/zcode-protocol/index.ts#L2762 (3.17.1)]` | `workspaceKey: nonEmptyString` — required in the session-create workspace shape |
| `[V-SRC packages/shared/src/zcode-protocol/index.ts#L2784 (3.17.1)]` | `workspaceKey: nonEmptyString.optional()` — optional variant shape |
| `[V-SRC packages/shared/src/zcode-protocol/index.ts#ZCODE_SESSION_RUNTIME_PREFERENCES_REQUEST_TIMEOUT_MS (3.17.1)]` | 15 s host-reply deadline for `session/requestRuntimePreferences` |
| `[V-SRC packages/shared/src/official-mcp-auth.ts#L149 (3.17.1)]` | `ZCODE_WORKSPACE_IDENTITY_ENV = "ZCODE_WORKSPACE_IDENTITY"` |
| `[V-SRC apps/zcode-cli/packages/bootstrap/src/zcode-protocol-entrypoint.ts#L219-L236 (3.17.1)]` | derivation convention `workspaceIdentity?.trim() || workspacePath` (verbatim quote above) |
| `[V-SRC packages/desktop/scripts/mcp-apps-host-fixture/agent.ts#L82-L86 (3.17.1)]` | reference host fixture: minimal `session/requestRuntimePreferences` answer |
| `[V-EXP EXP-WORKSPACEKEY-01]` | `experiments/captures/EXP-WORKSPACEKEY-01.jsonl` — both create verdicts, lifecycle churn, host-obligation discovery, hostSent replies |

---
*Phase: 1 — Research & Protocol Documentation (DISC-04)*
*Researched: 2026-11-19*
