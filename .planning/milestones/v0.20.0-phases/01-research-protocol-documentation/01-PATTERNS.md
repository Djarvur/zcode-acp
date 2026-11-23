# Phase 1: Research & Protocol Documentation - Pattern Map

**Mapped:** 2026-11-19
**Files analyzed:** 16 (11 authored files + 5 capture artifacts)
**Analogs found:** 6 / 11 authored files (all six docs; none of the five scripts — greenfield, see No Analog Found)

## Greenfield Status

Tracked content is exactly 34 files: `.claude/CLAUDE.md`, `.gitignore`, `.planning/**`, `LICENSE` (Apache-2.0), `README.md` (2 lines). **No source code exists.** The planning research documents ARE this phase's codebase. Every analog path below was verified with `git ls-files` (tracked-source gate); no gitignored mirrors exist in this repo.

## File Classification

| New File | Role | Data Flow | Closest Analog (tracked) | Match Quality |
|----------|------|-----------|--------------------------|---------------|
| `docs/protocol/README.md` | documentation (index + provenance legend) | static reference | `.planning/research/SUMMARY.md` (header block) | role-match |
| `docs/protocol/app-server-dialect.md` | documentation (claim-ledger) | static reference | `.planning/research/PITFALLS.md` (claim structure + inline citations) | role-match |
| `docs/protocol/acp-client-dialect.md` | documentation (claim-ledger) | static reference | `.planning/research/PITFALLS.md` + `.planning/research/REFERENCE-ADAPTERS.md` (neutral-wording precedent) | role-match |
| `docs/protocol/install-discovery.md` | documentation (claim-ledger, matrix) | static reference | `.planning/research/PITFALLS.md` | role-match |
| `docs/protocol/workspace-key.md` | documentation (claim-ledger) | static reference | `.planning/research/PITFALLS.md` | role-match |
| `docs/research/reference-adapters.md` | documentation (comparison ledger) | static reference | `.planning/research/REFERENCE-ADAPTERS.md` | exact |
| `experiments/README.md` | documentation (ops guide) | static reference | `.planning/research/SUMMARY.md` (header conventions) | role-match |
| `experiments/app-server-probe.mjs` | utility (experiment harness) | event-driven (stdio child, request-response JSONL) | none tracked — pattern source: `01-RESEARCH.md` lines 329–354 | no analog |
| `experiments/scrub-capture.mjs` | utility (transform) | file-I/O transform | none tracked — rules: `01-RESEARCH.md` lines 181–188 | no analog |
| `experiments/check-docs.sh` | utility (validation) | batch | none tracked — spec: `01-RESEARCH.md` §Validation Architecture (lines 509–520) | no analog |
| `experiments/scan-secrets.sh` | utility (security gate) | batch | none tracked — shape: `01-RESEARCH.md` lines 428–435 | no analog |
| `experiments/captures/EXP-{SUBSCRIBE,TURN,WORKSPACEKEY,DISCOVERY,SESSIONDIR}-01.jsonl` (5) | data artifact (Phase 3 fixture seed) | recorded output | none — format: `01-RESEARCH.md` line 188 | no analog |

## Pattern Assignments

### The five claim-ledger docs (all `docs/protocol/*.md` + `docs/research/reference-adapters.md`)

**Analog:** `.planning/research/PITFALLS.md` (structure), `.planning/research/REFERENCE-ADAPTERS.md` (provenance header + tables)

**Doc header block** (PITFALLS.md lines 1–7 — copy this shape):
```markdown
# <Title>

**Domain:** <one line>
**Researched:** 2026-11-19
**Confidence:** HIGH — <what was verified how>

Legend: **[V-EXP]** verified by experiment on the real machine · **[V-SRC]** verified by reading the source tree · ...
```
REFERENCE-ADAPTERS.md lines 3–10 shows the variant for docs whose claims start life as reported: `**Compiled:** <date>` + a `**Provenance:**` paragraph stating the label semantics and what Phase 1 re-verifies. Phase 1 docs state their own hierarchy per D-01: shipped 3.18.2 bundle = V-SRC ground truth; 3.17.x clone = corroborating, always version-cited.

**Inline claim tags** (PITFALLS.md line 18 shows the form; D-04 defines the Phase 1 grammar):
```markdown
...the registry keeps serving the *previous* snapshot [V-SRC: `packages/provider/src/registry-service.ts:205-212`] [V-EXP]
```
Phase 1 form: `[V-SRC <path>#<anchor>]`, `[V-EXP <experiment-id>]`, `[V-RPT <source>]`; every doc ends with a **Sources & provenance** section mapping its ids; the legend itself lives ONCE in `docs/protocol/README.md` and other docs reference it (checked by `check-docs.sh`).

**Per-doc required sections:** already blueprinted in `01-RESEARCH.md` §Pattern 1 (lines 164–170) — the planner should turn that nine-section list per doc into task actions; do not re-derive here.

### `docs/research/reference-adapters.md` (documentation, comparison ledger)

**Analog:** `.planning/research/REFERENCE-ADAPTERS.md` (exact — same role, same subject matter; the Phase 1 doc upgrades its V-RPT/V-SRC(parallel) claims to V-SRC)

**Adopt/avoid table shape** (lines 22–28):
```markdown
| Piece | What they do | Key file(s) |
|-------|--------------|-------------|
| GNOME Secret Service decryption | `~/.zcode/v2/credentials.json` (`enc:v2:os:`) via ... | `src/backend/credential-store.ts` |
```

**Attribution convention** (lines 14–15 — required section per DISC-05):
```markdown
TypeScript, npm `zcode-acp-server` (~0.88.x), **Apache-2.0** → porting code to
Go is allowed **with attribution** (keep NOTICE/license header in ported files).
```
This repo's own `LICENSE` is Apache-2.0; keep the attribution obligations consistent with it.

**Verdict vocabulary** (lines 33–41): "other adapters — for the record — all worse for our purpose" table with a `Why not` column; footer italic provenance line (`*...sources as listed.*`, line 136).

### `docs/protocol/README.md` (documentation, index + legend)

**Analog:** `.planning/research/SUMMARY.md` header block (lines 1–6):
```markdown
# Project Research Summary

**Project:** zcode-acp
**Domain:** Go stdio protocol adapter — ...
**Researched:** 2026-11-19
**Confidence:** HIGH — ...
```
Content: index of the six docs + the single provenance legend (D-04). Root `README.md` shows the house title style (`# zcode-acp` + one-line description).

### `experiments/app-server-probe.mjs` (utility, event-driven)

**No tracked analog exists** (no code in repo). Pattern source: `01-RESEARCH.md` lines 329–354 — a harness executed live against the real binary this session; copy this shape:
```javascript
// experiments/app-server-probe.mjs — pattern verified 2026-11-19 on 3.18.2
import { spawn } from 'node:child_process';

const [, , electronBin, cjs, builtinFile] = process.argv;
const child = spawn(electronBin, [cjs, 'app-server', '--stdio'], {
  env: { ...process.env,
         ELECTRON_RUN_AS_NODE: '1',
         ZCODE_BUILTIN_PROVIDER_CONFIG_FILE: builtinFile },
  stdio: ['pipe', 'pipe', 'pipe'],
});
child.stdout.on('data', d => appendRaw(d));           // raw file OUTSIDE the repo
...
setTimeout(() => child.stdin.end(), 12000);           // EOF pitfall: hold stdin open first
setTimeout(() => {                                    // backstop: kill process group
  try { process.kill(-child.pid, 'SIGKILL'); } catch {}
  process.exit(0);
}, 18000);

function send(obj) { child.stdin.write(JSON.stringify(obj) + '\n'); }
```
Load-bearing constraints (RESEARCH.md §Experiment Catalog, lines 207–213): (1) resolve ALL paths from CLI args — no machine-private paths hardcoded in committed scripts; (2) hold stdin open until collection window ends (EOF pitfall, Pitfall 1); (3) timestamp + collect stdout lines to raw file outside the repo; (4) mirror stderr capped; (5) `stdin.end()` for clean shutdown + process-group SIGKILL backstop.

### `experiments/scan-secrets.sh` (utility, batch gate)

**No tracked analog.** Pattern source: `01-RESEARCH.md` lines 428–435:
```bash
#!/usr/bin/env bash
# experiments/scan-secrets.sh — fail on any tracked file matching live-credential shapes
patterns=('eyJ[A-Za-z0-9_-]{10,}\.'      # JWT
          'enc:v[12]:os:'                 # credential blobs
          '"(access_token|apiKey|api-key|token)"\s*:\s*"[A-Za-z0-9_-]{20,}"')
rc=0; for p in "${patterns[@]}"; do
  git grep -nE "$p" -- . ':(exclude).planning' && { echo "SECRET SHAPE: $p"; rc=1; }
done; exit $rc
```

### `experiments/scrub-capture.mjs` (utility, transform)

**No tracked analog.** Rules defined once in `01-RESEARCH.md` lines 181–188 (Pattern 3): JWT shapes, `enc:v2:os:`/`enc:v1:` blobs, PAT/api-key shapes, UUID-v4/numeric account ids, user_info payloads, private-location paths. Capture file header (line 188) — the harness adds it, scrubber must preserve it:
```json
{"capture":"<experiment-id>","date":"...","desktopVersion":"3.18.2","builtinRevision":"<read at capture time>","host":"<os release>"}
```

### `experiments/check-docs.sh` (utility, batch)

**No tracked analog.** Spec = `01-RESEARCH.md` §Validation Architecture test map (lines 509–520): per-doc structure checks, V-tag coverage, `[V-EXP <id>]` → `experiments/captures/<id>*` resolution, legend-defined-once check, neutral-phrasing path-shape grep. Pure grep/jq, <30 s.

### `experiments/captures/*.jsonl` (data artifacts)

**No analog.** Named by experiment id (D-04: `[V-EXP <id>]` resolves to `experiments/captures/<id>*`): `EXP-SUBSCRIBE-01`, `EXP-TURN-01`, `EXP-WORKSPACEKEY-01`, `EXP-DISCOVERY-01`, `EXP-SESSIONDIR-01`. Contents per E1–E5 designs in `01-RESEARCH.md` §Experiment Catalog.

## Shared Patterns

### Provenance tagging (D-04)
**Source:** `.planning/research/PITFALLS.md:7` (legend form), `.planning/research/REFERENCE-ADAPTERS.md:3–10` (provenance header)
**Apply to:** all six docs — every claim line born with its tag; Sources & provenance section at end; legend defined once in `docs/protocol/README.md`.

### Neutral phrasing + no private paths
**Source:** `.claude/CLAUDE.md` constraints block (lines 20–27) via `.planning/PROJECT.md`; `01-RESEARCH.md` user_constraints
**Apply to:** every committed file — English only; no real machine paths of private research locations (patterns like `~/Applications/ZCode-*.AppImage` are fine, user-specific absolutes are not); hidden-goal wording forbidden — ACP-server phrasing stays neutral (REFERENCE-ADAPTERS.md §"Chosen preset-agent identity" is the tone precedent). Scripts receive private paths as runtime args.

### Secret hygiene (D-02)
**Source:** `01-RESEARCH.md` Pattern 3 + Pitfall 7; `.gitignore` (tracked, currently Go-oriented — has NO raw-capture entry; D-02's locked rule is raw captures born outside the repo, which is a discipline + scrub-first gate, not a gitignore entry — planner may still add `experiments/raw/` defensively)
**Apply to:** harness, scrubber, scan gate; scrubber is the only path into `experiments/captures/`.

### Commit conventions
**Source:** `.planning/config.json` (`commit_docs: true`; phase branch template `gsd/phase-{phase}-{slug}`)
**Apply to:** all Phase 1 artifacts are committed (including scrubbed captures — that is the point of D-02); user's global git workflow: fresh feature branch off up-to-date default branch.

### Ground-truth citation discipline (D-01)
**Source:** `01-RESEARCH.md` Pitfall 2 + §Code Examples (lines 371–378, 380–397)
**Apply to:** `app-server-dialect.md` — record WHICH bundle copy was probed (`cmp` vs `.orig`), cite 3.17.x source with version (`file:lines`), first-class 3.17↔3.18.2 drift-baseline section; occurrence counts via `grep -o | wc -l`, never `grep -c`.

## No Analog Found

| File | Role | Data Flow | Reason | Pattern Source Instead |
|------|------|-----------|--------|------------------------|
| `experiments/app-server-probe.mjs` | utility | event-driven | No tracked code exists (greenfield) | `01-RESEARCH.md` lines 329–354 (validated live) |
| `experiments/scrub-capture.mjs` | utility | transform | No tracked code exists | `01-RESEARCH.md` lines 181–188 (rule set) |
| `experiments/check-docs.sh` | utility | batch | No tracked code exists | `01-RESEARCH.md` lines 509–520 (test map) |
| `experiments/scan-secrets.sh` | utility | batch | No tracked code exists | `01-RESEARCH.md` lines 428–435 (shape) |
| `experiments/captures/*.jsonl` (5) | data artifact | recorded output | No fixtures exist | `01-RESEARCH.md` line 188 (header format) |

## Metadata

**Analog search scope:** all 42 git-tracked files; read in full or in part: `.planning/research/{REFERENCE-ADAPTERS,PITFALLS,SUMMARY}.md`, `.planning/ROADMAP.md`, `.planning/config.json`, `.gitignore`, `.claude/CLAUDE.md`, `README.md`, `LICENSE`; `01-CONTEXT.md` + `01-RESEARCH.md` (required reading).
**Files scanned:** 9 tracked analog candidates; 6 usable (all `.planning/` docs — no code analogs exist).
**Tracked-source gate:** passed — every analog path named above appears in `git ls-files`.
**Pattern extraction date:** 2026-11-19

## PATTERN MAPPING COMPLETE
