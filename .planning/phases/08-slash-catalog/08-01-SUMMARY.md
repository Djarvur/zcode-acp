---
plan: "08-01"
phase: "08"
slug: "slash-catalog"
---
# 08-01 — The slash catalog: `available_commands_update` on session/new

## Deliverable

Shipped (commit e18c527 + the fix-cycle commits): on every `session/new` the front
emits ONE `session/update` with `update.sessionUpdate = "available_commands_update"`
whose items are the ZCode commands (`~/.zcode/commands/*.md` + the `~/.agents/commands`
compat root; the item name is the file stem — the form the runtime provably invokes),
the ZCode skills (`~/.zcode/skills`, the `~/.agents/skills` compat root, the plugin
caches, and the session cwd's workspace pair; name/description from SKILL.md
frontmatter with the dir-name fallback), and the `/model` entry with the `<selector>`
hint. Deterministic order (commands → skills → model), silent skips for broken
manifests, never an error and never a hang — an emission failure logs and leaves
session/new successful.

## Wire truth chain (the user's directive: sources first)

- Installed-build chunks (the discovery driver `Jp`/`Yp`, the `IV` adapter, the `PV`
  mapper): the `available_commands_update` contract, the ≤9 s budget, the delete-vs-
  disposable-storage capability rule, the omp dialect's no-skillDescription mapping.
- orca v1.4.262 SOURCES (shallow clone, read fresh): `acp-at-rest-commands.ts`,
  `acp-session-discovery.ts` (the session-id-matched subscription, the capability
  check), `omp-native-commands.json` (the real omp item shape — input.hint), and the
  per-agent dialect table — every minified reading confirmed.
- zcode SOURCES (local clone): `skillsService.ts` (the four user/workspace roots +
  `normalizeSkillNameKey` lowercase dedupe + the dir-name fallback),
  `commandsService.ts` (the `~/.zcode/commands` + `~/.agents/commands` layout),
  `commandDataSync.ts` (the merge order — .zcode outranks), `skillDiscoveryWalk.ts`
  (the depth-8 recursive walk — our scan keeps the explicit glob shapes for
  determinism, documented).

## Verification

- Scanner/emitter unit + tracer locks: `TestSkillItems`,
  `TestSkillItemsDirNameFallback`, `TestDefaultCatalogRoots`, `TestCommandItems`,
  `TestCommandItemsRootPrecedence`, `TestDefaultCommandRoots`,
  `TestSessionNewEmitsCommandCatalog`, `TestSessionNewCatalogNoRoots`,
  `TestSessionNewEmitsCommandRoots` — RED→GREEN (the emission was compile-RED, the
  scanner contract RED via the broken-entry case).
- The pre-existing tracer/golden suite re-locked: the catalog frame joins the client
  stream — 4 goldens regenerated (the frame is line 1), the chunk-count contracts
  narrowed to chunk kinds (the catalog frame is a session-level nicety, not a turn
  update), `DrainUpdates` scoped to chunks.
- Full suite 10/10 packages under `-race`; whole-tree lint 0 issues.
- Live card-argv: a 139-item catalog on the real machine (the whole gsd-* set with
  argument-hints, the user skills, `/model`); turns unaffected (ok/done delivered).
- CAT-05 field pass: the VM GUI — the composer autocomplete renders `/model` + the
  seeded commands with hints (screenshot
  `experiments/captures/v021-catalog-vm-menu.png`); the user's real-machine look is
  the acceptance nicety (the binary is installed).

## Assumption outcomes

- The commands' invocation name = the file STEM (not the frontmatter `name:` —
  `gsd:manager` vs the proven `/gsd-manager`): CONFIRMED by the live probe.
- The omp skill rendering rides the same list as `command` kind items: CONFIRMED
  (the omp dialect defines no skillDescription).
