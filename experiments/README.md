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
      --window-ms 30000 [--quiet-ms 2500]

`--window-ms` must exceed the chosen flow's last scheduled step (the probe
validates this and refuses to start otherwise): the subscribe flow's final
probe fires at 13000 ms and the turn flow's at 13500 ms, so the example is
30000 — a shorter window lets the backstop win the race and the final probe
is never delivered (the run would still complete exit 0).

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

Flows: `probe` (spawn + provider registry warm-up), `workspacekey` (two
session/create calls with different workspaceKey shapes — no turn, no quota),
`subscribe` (E1 surface probe — no turn, no quota), and `turn` (E2 — the ONLY
quota-spending flow: one tiny real turn). The turn flow supports
`--end-mode quiet-window --quiet-ms 2500`: the run ends 2.5 s after the last
stdout line, gated by a send-latency floor and a settle gate (the close is held
until a `state.updated` patch reports a status other than "running" after the
send — a turn that never settles is collected until the `--window-ms` backstop).

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

## Run notes

### EXP-SUBSCRIBE-01 (E1, plan 01-02) — 2026-11-19

- Command shape: `node experiments/app-server-probe.mjs --electron <bundle-root>/zcode
  --cjs <bundle-root>/resources/glm/zcode.cjs --builtin <active zcode-builtin.json>
  --flow subscribe --raw-out <scratch-outside-repo>/raw-exp-subscribe-01.jsonl
  --window-ms 15000 --workspace-dir <scratch>/ws` — bundle root, entrypoint, and
  builtin config resolved at run time (real paths are supplied in the agent prompt,
  never committed); raw capture born outside the repo; scrubbed into
  `experiments/captures/EXP-SUBSCRIBE-01.jsonl` with `--redact-path` for the scratch
  prefix.
- Window: 15000 ms (research assumption A3 — wait longer than the 12 s probe).
- **Verdict (one line):** on shipped 3.18.2, `session/subscribe` **ACCEPTED**
  (result `{sessionId, eventSeq, events}`; eventSeq 0 with an empty event list for
  an idle no-send session), `session/events` **ACCEPTED** (poll-style, result
  `{events}`), `v4/conversation/subscribe` **REFUSED** (-32603) — its real params
  surface is `{topic: string, connectionId: string, clientMode:
  "desktop-continuous" | "web-remote-replayable"}` (the validation error lists
  `sessionId`/`deliveryKind` as unrecognized keys). No `state.updated` arrived
  within the window (A3 recorded honestly as not-observed here);
  `session/lifecycle/changed` envelopes carry `revision`,
  `startup/storageState` envelopes carry `sequence`. **No quota spent** — the flow
  contains no `session/send`.
- Bundle string context (occurrence counts via `grep -o … | wc -l`, Pitfall 8):
  `session/subscribe` x1, `v4/conversation/subscribe` x1, `session/events` x1,
  `deliveryKind` x61, `desktop-continuous` x15, `web-remote-replayable` x11,
  `session/delete` x0 (absent). The delivery-kind enum
  `["desktop-continuous","web-remote-replayable"]` appears in multiple minified
  schemas in the bundle — matching the 3.17.x `zcodeDeliveryKindSchema`.

### EXP-TURN-01 (E2, plan 01-02) — 2026-11-19

- Command shape: `node experiments/app-server-probe.mjs --electron <bundle-root>/zcode
  --cjs <bundle-root>/resources/glm/zcode.cjs --builtin <active zcode-builtin.json>
  --flow turn --raw-out <scratch-outside-repo>/raw-exp-turn-01.jsonl --window-ms 180000
  --end-mode quiet-window --quiet-ms 2500 --workspace-dir <scratch>/ws` — same
  path-resolution and scrub-first rules as E1; scrubbed into
  `experiments/captures/EXP-TURN-01.jsonl`.
- Outcome: **honest V-SRC-only degradation per D-05.** The turn flow works
  mechanically — create accepted, subscribe accepted, send accepted
  (`result{accepted, sessionId, stateRevision}`), the model request started and
  streamed — but no turn-completion envelope was ever observed on the wire: the
  session status patch stayed `"running"` until harness shutdown, and the client
  log recorded `turn.failed cancelled` at each teardown. The blocking host
  obligation is `interaction/requestCredentialCipher` (operation `decrypt`), a
  server→host request that fires mid-turn; answering it requires decrypted keyring
  credential material, which Phase 1 deliberately never reads (threat register
  T-1-05). Turn-completion terminal signals are therefore documented from bundle
  source enumeration + reported client-log facts; plan 01-03 labels the affected
  claims V-SRC-only / V-RPT.
- Live evidence captured anyway (the committed capture is not empty): turn-START
  signals — `computer-use/operation-event` with `kind:"turn-started"` + `turnId`,
  `state.updated` patch `{status:"running"}` reason `prompt_started`,
  `session/event` envelopes with `seq` numbering, `session/lifecycle/changed` with
  `revision`; and a live confirmation of the create-drops-reasoningLevel quirk —
  `session/setModel` refused `-32603 "Reasoning level is required for
  zai-api/GLM-5.3-Flash"` (the workaround needs `options.reasoningLevel`).
- Quota honesty note: three tiny `session/send` calls were issued across three
  runs (the first two were cancelled by harness shutdown after quiet-window
  mis-arms — both fixed in-plan and documented in the harness comments; the third
  by the same shutdown while the turn was still generating). No turn completed;
  no API key was minted; no CI involvement. See the quota ledger below.

## Quota ledger

- **real turns spent: 3 tiny sends issued, 0 completed** (plan 01-02, E2, 2026-11-19)
  — three `session/send` calls with the fixed one-line prompt; none produced a
  completed model response (see the EXP-TURN-01 run notes for the failure reason:
  the turn requires answering `interaction/requestCredentialCipher`, which Phase 1
  does not do). Plan 01-01 spent nothing (registry-level only). No key was ever
  minted; nothing here may ever run from CI.
- **Phase-5 guarded full-cycle run (plan 05-09, EXP-APIKEY-CYCLE-01) — PASSED 2026-11-21 (attempt 3): the full cycle completed on the real account through the drained test; actual spend the D-05 minimum**

  - **Attempt 1 (2026-11-21 00:00, orchestrator-run) — auth read-back FAILED, zero model spend.**
    Discovery OK (3.18.2 unpacked-appimage; our basedOn hash matched the rev-44 endpoint builtin file).
    provider/updateAccountConfig acked {status:"received", providerCount:0, receivedRevision:"account-bridge:<epoch-nanos>"};
    refreshRegistry {refreshed:true}; the desktop applied runtime headers for GLM-5.3-Flash in its own refresh loop —
    but no state.updated registry patch within the 30 s read-back window (the dialect §8 silent-discard shape).
    Zero turns. Capture: experiments/captures/EXP-APIKEY-CYCLE-01.jsonl (scan-secrets clean). Row stays PENDING for the passing run.
  - **DIAGNOSTIC (2026-11-21, plan 05-11, EXP-READBACK-DIAG-01) — branch verdict: (c) create-reply surface. ZERO model spend**
    (registry-level only: push → refresh → 30 s observation window → probe create → probe close; no turn ever started).
    Field facts: updateAccountConfig acked {status:"received", providerCount:0, revision echo verbatim}; refreshRegistry
    {refreshed:true}; NO state.updated in the 30 s window (startup/storageState ×5 only); probe session/create (temp-dir cwd)
    → settings.model.available carried zai-api/GLM-5.3-Flash + zai-api/GLM-5.3 (plus OpenCodeGo/* entries), lifecycle
    starting→ready, session/requestRuntimePreferences answered minimal, interaction/requestOfficialMcpAuthHeaders answered
    -32601; probe session/close → {closed:true}; watcher latched on the create surface; staged shutdown, zero orphans.
    Branch evidence: (c) — the create reply IS the registry read-back surface; the state.updated patch surface does not
    fire for push+refresh on desktop 3.18.2. Capture: experiments/captures/EXP-READBACK-DIAG-01.jsonl (scrubbed,
    scan-secrets clean). The EXP-APIKEY-CYCLE-01 row stays PENDING for the guarded REAL RUN re-run (05-12 human checkpoint).
  - **Attempt 2 (2026-11-21 09:38 UTC capture, human-run per the 05-12 checkpoint) — auth leg PASSED (read-back proven), tool-turn leg FAILED on the desktop turn latch. Real spend: one tiny completed turn.**
    Discovery 3.18.2 unpacked-appimage, rev-44 builtin matched; authenticate{zai-api-key} proven by the registry read-back verdict — the 05-11 branch-(c) fix is field-verified working
    (apicycle_test.go:205); session/new + D-03 subscribe gate + setModel zai-api/GLM-5.3-Flash; mid-turn cipher decrypt asks answered from the keyring (value_len=815, plaintext_len=555); the
    tiny tool-less turn COMPLETED end_turn with zero tool frames (apicycle_test.go:207) — the run's only model spend; the file-read tool turn was then refused by the desktop — "another prompt
    is already running (this agent runs one turn at a time)" (apicycle_test.go:208) → test FAIL at 10.34s; teardown read-loop warning benign (post-failure drain).
    Capture: experiments/captures/EXP-APIKEY-CYCLE-02.jsonl (scrubbed, scan-secrets clean; header V-EXP id EXP-APIKEY-CYCLE-01 is the test-locked captureID — file -02 preserves attempt-1's
    capture byte-intact). The row stays PENDING for a passing run; the phase stays gaps_found (next gap: the guarded test needs a turn-latch drain between sends).
  - **Attempt 3 (2026-11-21 19:29–19:30 UTC run, human-executed per the 05-14 checkpoint) — PASS 49.43s: hypothesis (a) VALIDATED, the full cycle completed on the real account through the drained test (05-13). Real spend: one registry read-back + two tiny completed turns (the D-05 minimum); no -32010 ever surfaced, so zero refused sends.**
    Same install (3.18.2 unpacked-appimage, rev-44 builtin matched); session scrubbed to <uuid-N> placeholders. authenticate{zai-api-key} proven by the registry read-back verdict; session/new + D-03 subscribe gate + setModel zai-api/GLM-5.3-Flash; mid-turn cipher asks answered from the keyring (79 encrypted values redacted by the scrub); tiny tool-less turn end_turn zero tool frames (our quiet-floor resolution 19:29:32.410Z per watcher.go floor math); the file-read tool turn ACCEPTED FIRST TRY through the drain — the settle budget (30 s) exhausted honestly with turn-1's cipher-ask loop still live (last ask 19:30:02.230Z, ~37 s total), the retry-phase send hit the wire 124 µs later (19:30:02.4736Z) and was accepted without any -32010; tool frames call_369286416d214a268f952583 pending→in_progress→completed (TOOL-01 live); session/resume same session no replay; session/close clean, post-close prompt rejected; staged shutdown zero orphans (kill(-pgid,0) ESRCH).
    (b)-flavored evidence recorded explicitly (the 05-14 branch rules' PASS clause): the desktop's TRUE turn-1 completion surfaces arrived ~16.4 s AFTER our quiet-floor end_turn — computer-use/operation-event kind=turn-completed at 19:29:48.740Z + state.updated rev 3 reason=prompt_completed at 19:29:49.093Z (attempt 2 observed NO terminal surface; both carriers exist on this run) — and turn-2's send found the latch already released 13.4 s after that patch; ask-silence does not correlate with latch release (the ask loop outlived the completion patch ~13 s). The PASS stands (the drain sufficed); this answers the §5 addendum's open question — terminal surfaces EXIST on 3.18.2, arriving ~16–17 s after the quiet floor — handed to the verifier/next gap cycle; no machine-side change rides this plan (zero-production contract; REQUIREMENTS.md untouched).
    Capture: experiments/captures/EXP-APIKEY-CYCLE-03.jsonl (NEW file — monotonic; EXP-APIKEY-CYCLE-01/-02 byte-intact; scrubbed 449/449 lines, scan-secrets exit 0 on the tree AND the staged diff, idempotent fixed point; header V-EXP id EXP-APIKEY-CYCLE-01 is the test-locked captureID). The row flips from PENDING on this passing run.
  (planned 2026-11-20; this row records the PLANNED spend — the end-of-phase
  verification run appends the actuals + verdict line beneath it)
  — planned spend (the D-05 minimum): one registry read-back (no turn) + two tiny
  turns (one tool-less, one exercising a file-read tool on a scratch file under the
  workspace) + the resume/close calls (no model spend). Tier: the guarded
  `test/realdesktop/apicycle_test.go` (`TestAPIKeyFullCycle`) — canonical command in
  its file header; NEVER from CI: the realdesktop build tag + `ZCODE_ACP_REAL_E2E=1`
  double guard is the contract, and the default suite never builds the file (the
  standing no-CI rule restated for the guarded tier — the account and the login
  keyring are manual-only). Evidence pointer format for the run's entry: scrubbed
  capture `experiments/captures/EXP-APIKEY-CYCLE-01.jsonl` (the raw capture from
  `ZCODE_ACP_RAW_CAPTURE` passes `scrub-capture.mjs` BEFORE any commit) + the V-EXP
  id + the actual spend. Credentials live only in the local ZCode desktop config
  (`ZCODE_ACP_API_KEY`, export into the manual-run shell only); the ledger records
  spend and V-EXP ids, never credentials.
- **Phase-6 guarded coding-plan full-cycle run (plan 06-08, EXP-CODINGPLAN-CYCLE-01) — PENDING (planned 2026-11-22; the HUMAN-GATED end-of-phase UAT — D-12/D-13; the executor never runs it)**
  — planned spend (the D-13 cap): one real PAT mint via the official api.z.ai chain
  (in-process only, never persisted) + the registry read-back (registry-level, no
  model spend) + ONE tool-less tiny turn — nothing else. Tier: the guarded
  `test/realdesktop/codingplan_test.go` (`TestCodingPlanFullCycle`) — canonical
  command in its file header (`export ZCODE_ACP_RAW_CAPTURE=<scratch path outside
  this repo>` then `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run
  TestCodingPlanFullCycle ./test/realdesktop/`); NEVER from CI: the realdesktop
  build tag + `ZCODE_ACP_REAL_E2E=1` double guard is the contract, and the default
  suite never builds the file. The credential is the desktop's own keyring login
  (D-12): NO key is exported, embedded, logged, or committed — the desktop must be
  logged into the individual coding plan and the session's keyring unlocked at run
  time. Evidence pointer format for the run's entry: scrubbed capture
  `experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl` (the raw capture from
  `ZCODE_ACP_RAW_CAPTURE` passes `scrub-capture.mjs` BEFORE any commit; scan-secrets
  green on the tree AND the staged diff) + the V-EXP id + the actual spend. The run
  confirms-or-adjusts the flagged assumptions A4 (settings.thoughtLevel in live
  replies), A6 (clientVersion tolerance in the mint), A7 (configOptions rendering
  when no current is reported) — outcomes recorded HERE and in COVERAGE.md's
  disposition column, never silently assumed. A failed leg: the honest verdict with
  the scrubbed capture; the phase stays open (the 05-14 discipline).
