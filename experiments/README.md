# Experiments — Phase 1 wire-truth pipeline

**Domain:** harness scripts, capture hygiene, and validation gates for the documentation phase
**Researched:** 2026-11-19
**Confidence:** HIGH — the harness pattern was executed live against the shipped 3.18.2 binary during phase research

This directory holds the experiment infrastructure for Phase 1 (per D-02/D-04):
a JSONL probe harness, the canonical capture scrubber, and the two validation
gates that run before every commit.

## Ground rules (read before running anything)

1. **Raw captures are born OUTSIDE the repo.** Point `--raw-out` at a scratch
   directory outside this repository. The scrubber is the ONLY path into
   `experiments/captures/` — never hand-edit or hand-create capture files.
2. **These scripts spawn the real desktop bundle and may spend account quota**
   (the turn flow sends a real prompt; every other flow is registry-level).
   NEVER wire them into CI — CI must never spawn the desktop or call the
   provider API. Only the two gates (`check-docs.sh`, `scan-secrets.sh`) are
   CI-safe tools.
3. **Machine paths stay on the command line.** The harness takes the bundle
   root, the entrypoint script, and the builtin config file as runtime
   arguments. No committed file may contain a user-specific absolute path —
   `check-docs.sh` fails the build on that shape.
4. **No account-state mutation.** No experiment in this phase mints API keys
   or writes account state; experiments are registry-level probes and, for
   the turn flow (added by plan 01-02), a single minimal prompt.

## Gates

    bash experiments/check-docs.sh               # structure + provenance-tag conformance of docs/
    bash experiments/check-docs.sh --phase-gate  # additionally: all six deliverable docs must exist
    bash experiments/scan-secrets.sh             # secret-shape gate over tracked files (before every commit)

`check-docs.sh` prints `CHECK FAIL: ...` for violations (exit 1) and
`PENDING: <path> (n/6 present)` for not-yet-written deliverable docs (affects
the exit code only under `--phase-gate`). `scan-secrets.sh` prints
`SECRET SHAPE: ...` and exits 1 when any tracked file outside `.planning/`
matches a live-credential value shape (three-part JWT tokens; credential-blob
payloads after the enc prefix; long token values under credential keys).
Documentation prose never trips the gate — the shapes match values, not
mentions.

## Harness: `app-server-probe.mjs`

    node experiments/app-server-probe.mjs \
      --electron <bundle-root>/zcode \
      --cjs <bundle-root>/resources/glm/zcode.cjs \
      --builtin <builtin-config-file> \
      --flow <flow-name> \
      --raw-out <scratch-path-outside-repo> \
      --window-ms 12000 [--quiet-ms 2500]

Spawns the app-server over stdio (`ELECTRON_RUN_AS_NODE=1`, builtin config
pointed at `<builtin-config-file>`), HOLDS STDIN OPEN for the whole collection
window (writing a request and closing stdin early makes the app-server exit
cleanly before answering — the drain window beats request processing; exit 0
plus an empty capture is the warning sign), writes timestamped stdout lines to
the raw file, mirrors child stderr capped (64 KiB), then shuts down cleanly
(`stdin.end()`) with a process-group kill backstop.

The first line of the raw file is the capture header JSON:

    {"capture":"<experiment-id>","date":"<ISO date>","desktopVersion":"3.18.2","builtinRevision":"<read from the builtin file at capture time>","host":"<os release>"}

The builtin revision is read at capture time (stale-evidence guard: a desktop
auto-update can swap the revision mid-phase, and the header makes that
detectable).

Flows: `probe` (spawn + provider registry warm-up) and `workspacekey` (two
session/create calls with different workspaceKey shapes — no turn, no quota).
Plan 01-02 adds the `subscribe` and `turn` flows at the dispatch seam.

## Scrubber: `scrub-capture.mjs` — the only path into captures/

    node experiments/scrub-capture.mjs <raw-capture> [output] [--redact-path <prefix>]...

Canonical rules (defined once here, enforced by `scrub-capture.test.mjs`):

- JWT-shaped values (three base64url segments) → redaction placeholder
- Credential-blob values — the `enc:v2:os:` prefix and the legacy `enc:v1:`
  form, each followed by a long base64ish body → fully redacted including
  the prefix
- Long token values under credential keys (`access_token`, `apiKey`,
  `api-key`, `token`) → redacted
- UUID-shaped identifiers → stable per-file placeholders: the nth distinct
  value becomes a deterministic nth placeholder, and the same value always
  maps to the same placeholder (referential integrity for Phase 3 golden
  fixtures)
- `user_info` payload objects → redacted wholesale
- Absolute paths under the user's home directory → tilde-relative form
- Path prefixes passed via repeatable `--redact-path` → `<path>`
  (private research locations are supplied at run time, never committed)

The header line passes through byte-identical; scrubbing is idempotent
(scrubbing already-scrubbed output changes nothing).

## Experiments

| Id | Goal | Flow / method | Quota |
|----|------|---------------|-------|
| `EXP-SUBSCRIBE-01` (E1, plan 01-02) | `session/subscribe` + event-stream surface verdict | `subscribe` flow | none |
| `EXP-TURN-01` (E2, plan 01-02) | turn-completion terminal signals | `turn` flow — one tiny turn | one turn |
| `EXP-WORKSPACEKEY-01` (E3, plan 01-01) | `workspaceKey` derivation on the wire | `workspacekey` flow | none |
| `EXP-DISCOVERY-01` (E4, plan 01-05) | install-layout enumeration | static probes + artifact inspection | none |
| `EXP-SESSIONDIR-01` (E5, plan 01-04) | session-dir format + model discovery | static chunk reads | none |

Invocation shape for every experiment: resolve the bundle root and builtin
config at run time; write the raw capture outside the repo; scrub into
`experiments/captures/<experiment-id>.jsonl`; run both gates.

## Quota ledger

- **Real turns spent: 0** — plan 01-02 updates this line; every experiment in
  plan 01-01 is registry-level and spends nothing.
