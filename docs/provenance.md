# Provenance and known limitations

**Audience:** users and integrators of the zcode-acp wrapper
**Scope:** what this binary promises about the ZCode desktop it drives, whose
code it builds on, and what it deliberately does not do
**Related:** the wire-level protocol notes under
[docs/protocol/](protocol/README.md) and the reference-implementation study in
[docs/research/reference-adapters.md](research/reference-adapters.md)

## Provenance statement

This wrapper is an honest adapter. The contract it keeps with every user:

- **The ZCode desktop bundle is never patched.** The wrapper drives the
  installed app-server over the app-server's own JSONL dialect and serves ACP
  v1 on its own stdio; no file of the desktop install is modified, no update
  or revision bookkeeping is forged, and the builtin-provider revision is
  computed from the install's real files.
- **No attestation or signature forgery.** Where a fact cannot be known, the
  wrapper reports nothing rather than a wrong number. The clearest case: the
  account quota is not exposed by the API over the OAuth coding-plan chain,
  so no quota counter is ever shown — this is a designed omission, not a
  missing feature. Nothing in the wrapper fabricates desktop attestations,
  signatures, or version identities.
- **The coding-plan PAT is minted through the official api.z.ai chain,
  exactly as the desktop's own binary signs it** — the same endpoints, the
  same client-type and client-version semantics (the desktop version is read
  from the real install tree; a documented `0.0.0` fallback covers installs
  without a version-looking path segment). The minted token is cached
  in-process and never logged or persisted.
- **The ACP surface stays clean.** Clients see ACP v1 frames only; the
  app-server's private dialect never leaks through. Errors that reach the
  client are coded, actionable, and honest — including the four keyring
  sentinels documented in the README's troubleshooting section.
- **The provenance label system** (`V-SRC` / `V-EXP` / `V-RPT`) used across
  the protocol notes is defined once in
  [docs/protocol/README.md](protocol/README.md) — that legend is the single
  definition site; this document only references it.

## Attribution

Portions of this project are ported from
[william0wang/zcode-acp](https://github.com/william0wang/zcode-acp)
(Apache-2.0), studied at tag v0.89.0. The ported surface is the DISC-05
verified set:

- the **v2-keyring credential chain** (Secret Service lookup, the v2 cipher
  envelope, the legacy v1 fallback ladder) — `internal/interact/credstore.go`;
- the **credential cipher serving** (the request/decrypt semantics the
  app-server's turn loop relies on) — `internal/interact/cipherdecrypt.go`;
- the **account-provider snapshot push** (the `zcode-builtin:<rev>:<hash>`
  revision formula) — `internal/hostsvc/snapshot.go`;
- the **coding-plan PAT flow** (the mint chain and its cache semantics) —
  `internal/hostsvc/patminter.go`.

The proven behaviors and formulas are ported; the TypeScript implementation
is not. Every ported Go file carries an in-file Apache-2.0 attribution header
naming the upstream project and tag. The full attribution record — upstream
license verification, the exact preservation obligations, and the studied
release — lives in
[docs/research/reference-adapters.md](research/reference-adapters.md)
(see its "Attribution" section). Apache-2.0 is permissive: no reciprocal
obligations beyond the notices above.

## Known limitations

Stated plainly, so nobody discovers them as failures:

- **No quota counter over OAuth.** The API does not expose the account quota
  on the coding-plan chain; the wrapper reports nothing rather than wrong
  numbers. Out of scope by requirement, permanently for v1.
- **Start Plan does not work headless.** Start Plan providers authenticate
  with a browser/captcha step only the desktop app can provide — the desktop
  login is the credential, and a fresh login or captcha refresh can never
  happen headless. The GLM Coding Plan works headless.
- **v1 supports Ubuntu GNOME only.** The credential paths need the Secret
  Service keyring (`secret-tool` from `libsecret-tools`, plus gnome-keyring).
  Other environments are accepted-post-v1 work.
- **Session list/load/fork real support is v2.** Today `session/delete` is
  answered as a graceful no-op and `session/fork` is declined with the
  standard method-not-found frame — spec-loose clients keep working, and the
  capability-gated client never sends them. These are designed decline arms,
  never crash paths.
- **The client's `agentCmdOverrides` persistence path is Settings-UI-only.**
  Configure the launch recipe through the client's Settings UI; where the
  client persists the override underneath is a labeled known-unknown
  (confirmed at the manual QA-03 run). See the README's launch recipe.

## The protocol notes

The wire-level ground truth behind the claims above — each a DISC-01..04
deliverable of the documentation phase:

| Document | Subject |
|----------|---------|
| [app-server-dialect.md](protocol/app-server-dialect.md) | the app-server's JSONL event surface: transport, method map, snapshot semantics, error codes |
| [acp-client-dialect.md](protocol/acp-client-dialect.md) | the client's ACP dialect: preset-agent identity, launch contract, version floor, string contracts |
| [install-discovery.md](protocol/install-discovery.md) | desktop install layouts and the discovery order the wrapper drives |
| [workspace-key.md](protocol/workspace-key.md) | `workspaceKey` derivation semantics |
| [reference-adapters.md](research/reference-adapters.md) | the reference-implementation study, adopt/avoid verdicts, and the attribution record |

---
*Created: 2026-11-22 (Phase 7 — DOCS-03). Facts only from shipped behavior;
the honesty contract it states is enforced in code, not aspirational.*
