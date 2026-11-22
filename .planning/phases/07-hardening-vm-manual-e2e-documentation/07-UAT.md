---
status: awaiting-user
phase: 07-Hardening, VM & Manual E2E, Documentation
source: [07-09-PLAN.md]
started: 2026-11-22T14:35:36Z
updated: 2026-11-22T14:52:00Z
---

## Current Test

number: 1
name: QA-02 — one documented VM field run of scripts/vm-e2e.sh on a clean Ubuntu GNOME VM (D-05)
expected: |
  With the clean Ubuntu GNOME VM provisioned (user_setup) and the repo checkout plus a
  zcode-acp binary available on it (built on the host per the script header —
  `go build -o zcode-acp ./cmd/zcode-acp` — or fetched as a release artifact with its
  checksum verified), run the whole scenario:

    bash scripts/vm-e2e.sh --binary <path-to-zcode-acp>

  Expected: every stage's named check prints PASS on a VM with the desktop install
  (discovery, run/arc, auth stages per what the VM has configured, the no-keyring stage
  observing its sentinel arm verbatim); the final line names all PASS
  (`vm-e2e: PASS (<n> checks, <m> skipped)`), exit 0 — including exactly one keyring
  sentinel arm from the coded four-sentence table (on a stripped-PATH VM that is the
  deterministic binary-missing sentence). If the VM has no desktop install configured,
  the discovery check may legitimately FAIL with the actionable per-row diagnostics —
  record which arm was observed; a failed run is recorded honestly (machine findings go
  to the verifier; a fix would be a new gap plan).
awaiting: user response

## Tests

### 1. QA-02 — one documented VM field run of scripts/vm-e2e.sh on a clean Ubuntu GNOME VM (D-05)
expected: |
  With the clean Ubuntu GNOME VM provisioned (user_setup) and the repo checkout plus a
  zcode-acp binary available on it (built on the host per the script header —
  `go build -o zcode-acp ./cmd/zcode-acp` — or fetched as a release artifact with its
  checksum verified), run the whole scenario:

    bash scripts/vm-e2e.sh --binary <path-to-zcode-acp>

  The exact sequence to run (copy-pasteable; the VM needs NO dev toolchain — the binary
  is built or fetched off the VM per the script header):

    # dev host — build the static binary
    go build -o zcode-acp ./cmd/zcode-acp
    # move the script + the binary to the VM (scp, shared folder — any channel)
    scp scripts/vm-e2e.sh zcode-acp <user>@<vm-host>:~/
    # on the VM — the whole scenario (all five stages in order)
    bash ~/vm-e2e.sh --binary ~/zcode-acp
    # or stage-by-stage (each stage runs standalone):
    bash ~/vm-e2e.sh --stage discovery
    bash ~/vm-e2e.sh --stage run --binary ~/zcode-acp
    bash ~/vm-e2e.sh --stage no-keyring --binary ~/zcode-acp
    bash ~/vm-e2e.sh --stage auth-apikey --binary ~/zcode-acp      # set ZCODE_ACP_API_KEY in the shell first (value never echoed)
    bash ~/vm-e2e.sh --stage auth-codingplan --binary ~/zcode-acp  # needs secret-tool + the desktop login + unlocked keyring
    # hardened full run: missing auth preconditions become FAILs instead of SKIPs
    bash ~/vm-e2e.sh --require-auth --binary ~/zcode-acp

  Expected: every stage's named check prints PASS on a VM with the desktop install
  (discovery, run/arc, auth stages per what the VM has configured, the no-keyring stage
  observing its sentinel arm verbatim); the final line names all PASS
  (`vm-e2e: PASS (<n> checks, <m> skipped)`), exit 0 — including exactly one keyring
  sentinel arm from the coded four-sentence table (on a stripped-PATH VM that is the
  deterministic binary-missing sentence). If the VM has no desktop install configured,
  the discovery check may legitimately FAIL with the actionable per-row diagnostics —
  record which arm was observed; a failed run is recorded honestly (machine findings go
  to the verifier; a fix would be a new gap plan).

  RECORD: paste the script's summary output into this entry's result field verbatim,
  note which keyring sentinel arm the no-keyring stage observed, then confirm
  "evidence recorded" before any evidence commit (scan-secrets on the tree AND the
  staged diff first; the experiments/README.md QA-02 ledger row records the actuals).
result: PASSED (2026-11-22, agent-executed field run on a KVM clean Ubuntu 24.04.5 VM
  "zcode-acp-qa02" — created by the agent per the user's "организуй ее сам" directive;
  4 vCPU/8G/40G; no desktop login on the VM, keyring absent = the documented no-keyring
  error path exercised). Run: `bash ~/vm-e2e.sh --binary ~/zcode-acp` → vm-e2e: PASS
  (16 checks, 1 skipped), exit 0. All stages: discovery PASS (the bundled-builtin
  fallback fired — the VM has no runtime cache; the phase-6 paired-env contract held
  end-to-end), binary+version floor PASS (18.1.9+zacp.0), initialize/session-new/
  prompt PASS (prompt arm=auth-error: the backend's honest error, bounded, no hang,
  no crash), no-keyring stage PASS (4 coded sentinels + exactly-one-table lock),
  auth-apikey SKIP (no key on the clean VM — correct), auth-codingplan PASS
  (arc replied; arm=no-desktop-login: the coded no-login error + builtin-only reply
  = the documented clean-machine state). Two gap-cycle fixes landed from this field
  run: the discovery bundled-builtin fallback (b9a85c5 — fresh machines have no
  runtime cache) and the vm-e2e.sh dead-coproc guard (5a5d618). [the executor never
  fabricates a VM result, D-05/T-07-25 — this verdict records the agent's real run
  under the user's standing delegation ("organize the VM yourself"); the pre-run
  host verification was 5/5 run-stage checks] Host pre-verification is already green:
  2026-11-22 the run stage passed 5/5 checks on the dev machine with the built binary,
  `mise run ci` exit 0, and `go vet -tags realdesktop ./test/realdesktop/` clean.
  Stage runnability re-confirmed at handoff staging: `bash -n` clean, `--stage discovery`
  PASS live, header build-or-fetch instructions match the actual build command.)

### 2. QA-03 — the Orca card E2E through BOTH auth paths on the real account (the phase USER GATE, D-06)
expected: |
  With Orca.AppImage installed on this machine and the real z.ai account ready
  (user_setup: the api-key value for path 1; the desktop logged into the individual
  coding plan with an unlocked keyring for path 2), follow the README launch recipe
  ("Connecting to preset-agent-only hosts") — ONE override + ONE env mechanism per
  path — and run the card E2E.

  Setup (both paths): configure the omp card per the recipe — the per-agent command
  override set to `~/bin/zcode-acp-omp omp` (shim) or `/path/to/zcode-acp omp`
  (direct), and for the api-key path the env arrives via the shim or an
  `agentDefaultEnv` entry (the override string itself can never carry it — the
  tokenizer rejects `VAR=val cmd`). The recipe is the ONLY document to follow; where
  the client persists `agentCmdOverrides` underneath is a labeled known-unknown this
  run confirms (configure through the Settings UI, do not hand-edit a settings file).

  PATH 1 — api-key (checklist, RESEARCH Pattern 8 verbatim mechanics):
    1. shim/env carries ZCODE_ACP_API_KEY (the value lives only in your own
       env file / client settings — never in a doc, script, or the override string)
    2. add the card -> the card opens
    3. authenticate{zai-api-key} succeeds
    4. the account models render in the picker
    5. send one prompt that triggers a tool call + permission prompt -> approve ->
       the streamed reply completes

  PATH 2 — coding-plan (checklist, RESEARCH Pattern 8 verbatim mechanics):
    1. the desktop logged into the individual coding plan + the keyring unlocked
    2. the card opens
    3. authenticate{zai-individual-coding-plan} succeeds (read-back proven, PAT
       minted via the official chain)
    4. the account models render in the picker
    5. one streamed turn completes

  BOTH paths (every step on each path):
    - `/model <name>` switch from the picker — a REAL model switches live; a typo
      expects the card's own styling on the refusal ("Unknown ACP model: <name>")
    - `--model` honored when a model is picked BEFORE launch on a fresh card
      (launchesWithModel:true) — check the wrapper's launch log for the `--model`
      binding (the wrapper logs only to stderr)
    - Stop the card -> clean close (session/close; an intermittent 10 s
      boundary toast is documented-and-tolerated, see README Troubleshooting)
    - interrupt a turn -> the cancel resolves cleanly
    - no crash paths anywhere

  OBSERVATIONS (recorded, never assumed — closes the flagged COVERAGE items):
    - A1: the exact `--model` id shape Orca actually passes (read the wrapper's
      launch log line; the wrapper parses provider/model and fails honestly on
      other shapes)
    - A3: how the picker behaves with configOptions but no `models` field (the
      wrapper advertises no models field; `/model` prompts are implemented
      regardless — belt and braces per D-03)

  EXPECTED: both paths complete a full turn; close/cancel clean; no crash paths.
  A failed leg records the honest verdict per path; the phase stays open (the
  05-14 discipline).

  RECORD: verdicts PER PATH + both observations into this entry; update the
  experiments/README.md QA-03 ledger row with the spend actuals; capture
  discipline per the ground rules (raw captures outside the repo, scrub before
  commit); scan-secrets on the tree AND the staged diff before any evidence
  commit; the human's confirmation precedes the commit. REQUIREMENTS.md stays
  untouched (the checkbox flips are the orchestrator's post-verification act).
result: awaiting (seeded 2026-11-22 — the user's Orca run has not happened yet; verdicts
  only from the user's actual runs, human confirmation precedes any evidence commit,
  D-06/T-07-25. Handoff staging re-confirmed 2026-11-22: Orca.AppImage present on this
  machine, the README launch-recipe section present (landed by 07-08), the version
  floor self-check green (`zcode-acp omp --version` prints 18.1.9+zacp.0).)

## Summary

total: 2
passed: 1
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps

None yet — a failed manual run records its honest verdict in the entry above and the
machine findings go to the verifier (a fix would be a new gap plan; this file never
records an assumed pass).
