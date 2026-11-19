# Walking Skeleton — zcode-acp

**Phase:** 1
**Generated:** 2026-11-19

## Capability Proven End-to-End

> One wire-truth claim travels the full documentation pipeline: source citation → experiment run
> against the shipped 3.18.2 app-server → scrub-first capture committed under `experiments/captures/`
> → provenance-tagged claim lines in `docs/protocol/` → `check-docs.sh` + `scan-secrets.sh` green.
> Phase 1 proves the pipeline on the smallest deliverable (`workspaceKey` semantics, plan 01-01);
> every other deliverable in the phase rides the same pipeline unchanged.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Documentation framework | Claim-ledger Markdown under `docs/protocol/` + `docs/research/`, one doc per success criterion (D-03) | Later phases (3–7) quote these docs in their success criteria; every claim is born with its provenance tag |
| Provenance system | Inline tags `[V-SRC <path>#<anchor>]` / `[V-EXP <id>]` / `[V-RPT <source>]`; legend defined exactly once in `docs/protocol/README.md` (D-04); `[V-EXP <id>]` resolves to `experiments/captures/<id>*` | One legend location, machine-checkable id resolution via `check-docs.sh` |
| Ground-truth hierarchy | Shipped unpacked 3.18.2 bundle = V-SRC ground truth; 3.17.x source clone = corroborating, always version-cited; first-class drift-baseline section in the dialect doc (D-01) | Never present 3.17.x-source behavior as 3.18.2 truth |
| Experiment harness | Node `.mjs` spawn scripts under `experiments/` driven by `--flow` names; ALL machine paths arrive as CLI args, never hardcoded | stdin-hold timing + process-group backstop proven live against the real binary; private locations stay out of committed files |
| Capture pipeline | Scrub-first: raw captures are born outside the repo; `experiments/scrub-capture.mjs` is the ONLY path into `experiments/captures/`; scrubber replaces identifier shapes with stable placeholders to keep referential integrity for Phase 3 golden fixtures (D-02) | Committed captures are history-rewrite territory (costly) if scrub rules miss; one canonical scrubber = auditable rules |
| Validation | Shell gates: `experiments/check-docs.sh` (structure + tag grammar + V-EXP id resolution + path-shape grep; `--phase-gate` requires all six docs) and `experiments/scan-secrets.sh` (secret VALUE-shape gate; `.planning` excluded) | <30 s feedback after every task commit; no test framework exists (greenfield docs phase); Phase 2 CI reuses `scan-secrets.sh` |
| Secrets policy | Scan regexes match secret VALUES (long tails after JWT/blob/token shapes), never bare prefixes; test fixtures build forbidden shapes by string concatenation | Docs legitimately name blob shapes; only secret values are forbidden in tracked files |
| Language / phrasing | English only; neutral ACP-server phrasing; no private machine paths in any committed file — real paths are supplied in execution agent prompts only | PROJECT.md constraints; hidden-goal rule |

## Stack Touched in Phase 1

- [ ] Documentation — six deliverable docs (five under `docs/protocol/`, one under `docs/research/`)
- [ ] Experiments — `app-server-probe.mjs` harness + E1–E5 runs + five scrubbed captures
- [ ] Validation — `check-docs.sh` + `scan-secrets.sh` + scrubber unit tests (`node --test`, built-in runner)
- [ ] Local-run command documented — `experiments/README.md` with the never-in-CI warning and the quota ledger

## Out of Scope (Deferred to Later Slices)

- Any wrapper/Go code (Phase 2+); CI wiring of the gates (Phase 2 adopts `scan-secrets.sh`, never the experiment scripts)
- Live chat-UI card integration (Phase 7 manual E2E); no live card run in Phase 1 (D-06)
- Auth implementation (Phases 5–6); minting API keys; patching the ZCode bundle (never — honesty posture)

## Subsequent Slice Plan

- Phase 2: repo scaffold + CI; `scan-secrets.sh` becomes a CI job (experiment scripts stay manual-tier, never CI)
- Phase 3: codec / fake app-server / supervisor built from `app-server-dialect.md` + `install-discovery.md`; scrubbed captures become golden fixtures
- Phase 4: minimal ACP chat loop from the terminal-signal table (EXP-TURN-01) and the subscribe verdict (EXP-SUBSCRIBE-01)
- Phases 5–6: interaction shapes, api-key + OAuth chains per the dialect doc + `reference-adapters.md` adopt list
- Phase 7: hardening, VM scenario, manual E2E, English docs including the preset-agent launch recipe
