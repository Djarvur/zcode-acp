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
- **Phase-6 guarded coding-plan full-cycle run (plan 06-08, EXP-CODINGPLAN-CYCLE-01) — PASSED (2026-11-22 12:02 local, attempt 8; the HUMAN-GATED end-of-phase UAT — D-12/D-13; run by the orchestrator on the user's explicit "запускай" authorization)**
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
  - **Attempt 1 — FAILED at the registry read-back (2026-11-22 09:15–09:16 local, ZERO spend: no PAT mint, no turn — the chain aborted before the mint).** Keyring probe ok (13 ms); entitlement gate entitled=true for family=zai on the real setting.json; snapshot pushed with basedOnZCodeBuiltinRevision `zcode-builtin:44:e9c0674…` — ack `providerCount: 5, received`; refresh ok; probe session reached ready — but the create reply's `settings.model.available` carried ONLY the builtin registry, the account-provider models never surfaced, and the 30 s read-back window expired (`EXP-READBACK-DIAG-01` surface absent). Root cause found in the capture: the desktop fired `interaction/requestOfficialMcpAuthHeaders` (zcode-site plugin → https://zcode.z.ai) and our -32601 unknown-method error stalled the session's runtime materialization — the account models never entered the session registry. Wire truth from the 3.18.2 bundle: the real host answers this ask with the STRUCTURED RESULT `{ok:false, reason:"official_mcp_origin_untrusted"|"official_auth_unavailable"}` (survivable, plugin proceeds header-less) — never an error envelope. Fixed in-phase: honest-decline policy row + `official-mcp-auth-decline` corpus golden (`24063cf`, `a9000b6`); scrubbed attempt-1 capture committed as `experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl` (V-EXP attempt-1). Attempt 2 pending with the fix in the tree.

  - **Phase-7 QA-02 clean-VM field run (plan 07-07/07-09, EXP-VM-QA02) — PASSED (2026-11-22, agent-executed on the user's delegation).** VM: KVM clean Ubuntu 24.04.5 ("zcode-acp-qa02", 4 vCPU/8G/40G, no desktop login). Run: `bash vm-e2e.sh --binary ~/zcode-acp` → PASS (16 checks, 1 skip), exit 0 — install (desktop copy 846M)/discovery (bundled-builtin fallback: fresh machines have no `~/.zcode/v2/runtime` cache — discovery fix b9a85c5)/version floor 18.1.9+zacp.0/initialize/session-new (account-model read-back contract held on the paired env)/prompt (honest auth-error arm)/no-keyring (4 coded sentinels verbatim)/auth-apikey SKIP. Two gap-cycle fixes: discovery bundled-builtin fallback (b9a85c5) + vm-e2e.sh dead-coproc guard (5a5d618). Zero quota spend. Capture: `experiments/captures/EXP-VM-QA02.jsonl`.
  - **Attempt 8 — PASSED end-to-end (2026-11-22 12:02 local, 15.2 s).** authenticate ok (read-back proven, PAT mint-once); configOptions live (model + thought_level rendered from the bound account model); session/new bound `account:zai-individual-coding-plan/GLM-5.3-Flash` (reasoning `high`); the turn fired exactly ONE `interaction/requestProviderRuntimeHeaders` ask, answered with the three-field requestAuth shape (token material never recorded); `end_turn`, zero tool frames, staged shutdown. Assumption outcomes: **A4 CONFIRMED** (thought_level rendered live: currentValue "high", 3 entries); **A6 CONFIRMED** (the mint chain succeeded against api.z.ai through the configured proxy); **A7 CONFIRMED-SHAPE** (configOptions rendered the confirmed binding — currentValue carried the account model; entries reflected the bound-provider view). Evidence: scrubbed capture `experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl` (the PASS run), V-EXP id EXP-CODINGPLAN-CYCLE-01. The harness env for an account-model turn: `ZCODE_ACP_MODEL=account:zai-individual-coding-plan/GLM-5.3-Flash ZCODE_ACP_REASONING_LEVEL=high` (the PAT-serving ask fires only when the turn materializes an ACCOUNT model — the default zai-api binding skips it).
  - **Spend honesty (gap-cycle cost):** attempts 1–3 zero-spend (aborted before the mint). Attempts 4–8 each ran the round-3 EAGER PAT mint at authenticate start (up to 5 successful mint chains ≈ 4–5 api.z.ai call-clusters each; no model tokens anywhere — every turn attempt either never started or was the final 1-turn PASS). Total ≈ one D-13 planned cycle + 4–5 registry-level mint re-runs from the gap cycles; recorded here in full.
  - **Attempt 2 — FAILED at the same read-back, deeper reach (2026-11-22 09:44–09:45 local, ZERO spend: no PAT mint, no turn).** The attempt-1 fix held: `requestOfficialMcpAuthHeaders` answered `{ok:false, reason:"official_auth_unavailable"}` and the session materialized past it; two real keyring decrypts answered ok (context7 MCP OAuth metadata + a timestamp — the zcode-site plugin's stored credentials, NOT the zai account); two desktop encrypt asks (same values re-encrypted) answered the standing `unsupported-operation` decline — post-create, NOT on the read-back path (create reply frame precedes them). Create reply `settings.model.available` again builtin-only (zai-api pair + OpenCodeGo third-party) — account models absent. Root cause (wire truth from zcode-src 3.17.1, corroborating tier): the runtime's registry revision is `zcode-builtin:<rev>:<sha256(resolve(winnerPath))>` over a TWO-candidate selection — bundled (in-install) vs the desktop's active cache `~/.zcode/v2/runtime/provider/<platform>/<appVersion>/endpoint-*/zcode-builtin.json` — where TIES fall to BUNDLED (`selectReleaseCandidate`); on this machine both are rev 44 with different bytes, so the runtime's effective revision is the bundled-path hash `44:39ab3262…` while we pushed the active-path hash `44:e9c0674…` → `registry-service.ts:205` silently ignores the mismatched snapshot (schema-accepted, registry-ignored). Fix in flight: effective-revision resolver implementing the same two-candidate selection (attempt 3 pending). Scrubbed attempt-2 capture: `/tmp/zcode-acp-uat-10/codingplan-cycle-02.raw.jsonl` (persist on PASS or on final close-out).
  - **Attempt 3 — the tie→bundled hypothesis WITHDRAWN, its resolver REVERTED, before a real run (2026-11-22, ZERO spend: never wired against the runtime).** The "fix in flight" above (the effective-revision resolver implementing the 3.17.1 clone's tie→bundled `selectReleaseCandidate`, commits `925d3af`..`540550b`) would push the BUNDLED-copy hash `zcode-builtin:44:39ab3262…` — a string the shipped 3.18.2 runtime's registry provably does not carry: the desktop GUI host's own provider-runtime trace (`~/.zcode/v2/logs/2026-11-15.log`, the host running its OWN session on this machine: `[provider-runtime][trace:Provider Registry 已就绪]`) records `configRevision` `["zcode-builtin:44:e9c0674…", "<personal-hash>"]` — the ACTIVE-cache path hash — while a same-revision bundled copy (rev 44, different bytes, hash `44:39ab3262…`) exists on the same machine. The 3.17.1 clone's selection rule does NOT hold on the shipped runtime; the resolver left the tree (round-3 revert) and the attempts-1-2 revision string — the ACTIVE-cache hash, pushed verbatim — stands as correct. The ACTUAL attempts-1-2 root cause is elsewhere: the probe (session/create + close) never materializes the account provider — no `session/setModel` poke → no model-selection materialization → no `interaction/requestProviderRuntimeHeaders` ask → no PAT answer → the `account:zai-*` models never publish. Fix in flight (round-3): the coding-plan authenticate probe pokes the deterministic want model (the account family's `-Flash` preference) after create-ready, with the PAT minted eagerly before the probe so the materialization ask answers from the cache.
- **Phase-7 QA-02 VM field run (plan 07-09, one documented execution of `scripts/vm-e2e.sh` on a clean Ubuntu GNOME VM) — PENDING (seeded 2026-11-22; awaiting the user's VM run — D-05: the script is the deliverable, one documented user run is the field proof).** Recording shape: `.planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md` Test 1 (the script's summary output pasted verbatim; which keyring sentinel arm was observed; a failed run recorded honestly). Planned spend: ZERO API spend — discovery is read-only probing, the no-keyring stage asserts a coded sentinel (no network, no turn), the run/auth stages complete one minimal prompt only where the machine is already desktop-authenticated (an unauthenticated desktop answers the backend's honest auth error — bounded, no hang; auth stages SKIP without credentials). Host pre-verification already green (2026-11-22: run stage 5 PASS / 0 skipped on the dev machine; `mise run ci` exit 0). Evidence discipline: env-var NAMES only (`ZCODE_ACP_API_KEY` is never echoed); scan-secrets on the tree AND the staged diff before any evidence commit; the human's confirmation precedes the commit (T-07-25: verdicts only from real runs).
- **Phase-7 QA-03 Orca card E2E (plan 07-09, THE phase user gate, D-06) — CLOSED PASSED (2026-11-23; path 2 by the user's real run, path 1 accepted-closed by the user's recorded decision).** Recording shape: `.planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md` Test 2 — verdicts PER PATH plus the two observation items (the exact `--model` id shape Orca actually passes, A1 — read from the wrapper's launch log line; the picker's behavior with configOptions but no `models` field, A3), never assumed. Planned spend (the phase-5/6 D-05 minimum, per path): one registry read-back + two tiny completed turns — path 1 (`zai-api-key`): the authenticate read-back + one streamed turn with a tool call + permission prompt; path 2 (`zai-individual-coding-plan`): the keyring probe/PAT-mint authenticate read-back + one streamed account-model turn. Capture discipline per the ground rules above: raw captures born OUTSIDE the repo, the scrubber is the only path into `experiments/captures/`, scan-secrets on the tree AND the staged diff before any evidence commit, env-var NAMES only in recorded commands; the human's confirmation precedes the evidence commit (T-07-25).
- **Phase-7 QA-03 gap cycle 3 — the terminal-before-text race: CLOSED with agent-verified runs (2026-11-23 01:50–02:22 local, e30cb97).** The user's Orca card run (2026-11-22 evening) saw a prompt with NO response and NO errors. Local card-argv reproduction (the exact production invocation `zcode-acp omp acp --plugin-dir … --model zai-api/GLM-5.3-Flash`) reproduced it: thought chunks streamed, then `end_turn` with ZERO message chunks and the log `no-delta reconcile found no assistant text in the messages projection; replying empty`. Wire truth (fresh full-line raw capture `/tmp/qa03-raw5.out`): the desktop's reasoning deltas → text delta → turn-completed ordering is NORMALLY safe, but the delivery race is REAL run-to-run — the terminal signal can land BEFORE the reply text reaches the session/messages projection, and the reconcile's SINGLE poll read the not-yet-caught-up projection and gave up. Fix e30cb97: the reconcile retries the projection on a 400ms interval inside a 4s total window (never a hang; the message-count guard keeps late deltas from duplicating the recovered text — a late delta rides the same held pump). **Verification honesty note:** three interim "successful" runs were later found to have used the STALE installed binary (Nov 22 23:27, pre-fix) — they were the lucky side of the same race, NOT proof; `go install` from the fixed tree (02:20) preceded the final evidence. Final evidence on the FIXED binary: 4 consecutive two-prompt same-session card-argv runs (8 turns) — EVERY turn delivered its text chunk (`ok`/`done`) before `end_turn`, zero empty replies, zero reconcile-failure warnings. The user's GUI confirmation pass (QA-03 Test 2 verdicts) remains open per T-07-25. **Final spend actuals (the whole QA-03 arc, 2026-11-22/23): ~20 tiny real-account turns (one-word replies, no tool calls except the /gsd-manager probe's two Read calls; no PAT re-mints beyond the phase-6 mint-once; registry-level reads otherwise) + ZERO-spend fake-tier runs (the VM GUI pass, the corpus replays). Evidence: 07-UAT.md Test 2 (PASSED), the delta-review disposition (4 resolved in-phase 8515654), the verification report 07-VERIFICATION.md (passed 36/36).
- **QA-03 GUI-in-VM rendering pass (2026-11-23 02:35–03:00 local, agent-executed per the user's explicit delegation "рендеринг ты можешь проверять в виртуалке" — the rendering check is the agent's eyes, in a VM).** Composition: the qa02 VM (GNOME-capable clean machine) + Orca (fresh profile, agent added through its own onboarding UI driven over CDP) + the omp card pointing at the FIXED wrapper (`/usr/local/bin/omp` shim → `zcode-acp omp`) + the app-server override (ZCODE_ACP_APP_SERVER) feeding `cmd/fake-app-server` with a composed auth+streaming corpus scenario (auth-apikey legs + desktop-delta-streaming turn, capture-verbatim frames). VERIFIED IN THE GUI: the chat renders the user bubble, the Reasoning section (thought chunks) and the assistant reply text for BOTH turns of one session ("Answer: done" = the corpus deltas verbatim — proof by construction the frames crossed wrapper→Orca), and closing the chat tab terminates the wrapper process cleanly (no orphans, no crash). Screenshots: `experiments/captures/qa03-gui-vm-two-turns.png` (both turns + expanded Reasoning). SPEND: ZERO API (fake tier; the real-model leg was already terminal-verified 8/8 on the host). Scope honesty: this pass proves the wrapper→card rendering surface end-to-end; the real-account picker/model-switch items (real desktop login) remain tied to the user's machine and stay in the UAT checklist. Spend honesty: ~14 tiny one-word turns on the real account across the debugging + verification runs (no tool calls, no PAT re-mints, registry-level only); captures born in /tmp, nothing committed unscrubbed.
