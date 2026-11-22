---
status: awaiting-user
phase: 07-Hardening, VM & Manual E2E, Documentation
source: [07-09-PLAN.md]
started: 2026-11-22T14:35:36Z
updated: 2026-11-22T14:47:30Z
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
result: awaiting (seeded 2026-11-22 — the user's VM run has not happened yet; the executor
  never fabricates a VM result, D-05/T-07-25. Host pre-verification is already green:
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
  path — and run the card E2E:

  - PATH 1 (api-key): add the card -> the wrapper authenticates zai-api-key -> account
    models render -> send one prompt that triggers a tool call + permission prompt ->
    approve -> the streamed reply completes.
  - PATH 2 (coding-plan): desktop logged in, keyring unlocked -> the card opens ->
    authenticate zai-individual-coding-plan (read-back proven, PAT minted) -> the
    account models render -> one streamed turn completes.
  - BOTH paths: `/model <name>` switch from the picker (a typo expects the card's own
    styling — "Unknown ACP model"); a model picked BEFORE launch is honored at session
    creation (check the wrapper's launch log for the --model binding); Stop closes the
    card cleanly; interrupting a turn resolves the cancel; no crash paths anywhere.

  EXPECTED: both paths complete a full turn; close/cancel clean; the two OBSERVATION
  items recorded — which id shape Orca actually passes as `--model` (A1, read from the
  wrapper's launch log line) and how the picker behaves with configOptions but no
  models field (A3). A failed leg records the honest verdict per path; the phase stays
  open (the 05-14 discipline).
result: awaiting (seeded 2026-11-22 — the user's Orca run has not happened yet; verdicts
  only from the user's actual runs, human confirmation precedes any evidence commit,
  D-06/T-07-25.)

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps

None yet — a failed manual run records its honest verdict in the entry above and the
machine findings go to the verifier (a fix would be a new gap plan; this file never
records an assumed pass).
