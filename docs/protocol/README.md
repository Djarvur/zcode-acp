# Protocol Documentation

**Domain:** ZCode desktop app-server wire-truth documentation — index and provenance legend
**Researched:** 2026-11-19
**Confidence:** HIGH — structure locked by phase decisions D-03/D-04; ground-truth hierarchy per D-01

Index of the documentation phase's claim-ledger documents. Every claim in these
docs carries an inline provenance tag; the legend below is the single definition
site (D-04) — other docs reference this README instead of redefining it.

## Deliverables

| Document | Requirement | Subject |
|----------|-------------|---------|
| [app-server-dialect.md](app-server-dialect.md) | DISC-01 | App-server JSONL event surface: transport facts, method map, `session/subscribe` availability, turn-completion terminal signals, error codes, session lifecycle, snapshot semantics, drift baseline |
| [acp-client-dialect.md](acp-client-dialect.md) | DISC-02 | Client ACP dialect: preset-agent identity, version floor, launch contract, load-bearing method set, full method range, string contracts |
| [install-discovery.md](install-discovery.md) | DISC-03 | Desktop install discovery matrix: AppImage / deb / unpacked layouts, auto-update effects on the builtin revision |
| [workspace-key.md](workspace-key.md) | DISC-04 | `workspaceKey` derivation semantics: wire schema, convention, experiment record, host-freedom note |
| [../research/reference-adapters.md](../research/reference-adapters.md) | DISC-05 | Reference-implementation comparison: coverage, adopt/avoid verdicts, attribution obligations |

Supporting artifacts: the `experiments/` harness and validation gates (ops guide
in `experiments/README.md`), and scrubbed wire captures under
`experiments/captures/` — provenance experiment ids resolve there.

## Provenance legend

Every claim carries exactly one inline tag:

- `[V-SRC <path>#<anchor>]` — verified by reading source. Claims about the shipped
  desktop bundle cite the bundle copy actually probed (ground truth per D-01; the
  pristine byte-compare against the unmodified backup is recorded in the experiment
  record). Claims citing the public 3.17.x source tree always carry the version
  they were read at (e.g. `3.17.1`) and count as corroborating evidence only.
- `[V-EXP <experiment-id>]` — verified by experiment on the running binary; the id
  resolves to a scrubbed capture file `experiments/captures/<experiment-id>*`,
  produced only by the committed scrub-first pipeline (raw captures never enter
  the repository).
- `[V-RPT <source>]` — reported by a third party or inherited from earlier private
  research notes; not verified in this phase.

Ground-truth hierarchy (D-01): the **shipped 3.18.2 desktop bundle** is V-SRC
ground truth for shipped-binary behavior; the **public 3.17.x source tree** is
corroborating only and is never presented as 3.18.2 truth. Load-bearing claims
are additionally backed by an experiment tag where feasible.

---
*Phase: 1 — Research & Protocol Documentation*
*Created: 2026-11-19*
