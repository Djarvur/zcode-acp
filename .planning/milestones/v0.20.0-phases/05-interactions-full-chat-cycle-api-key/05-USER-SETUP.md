# Phase 5: User Setup Required

**Generated:** 2026-11-20
**Phase:** 05-interactions-full-chat-cycle-api-key
**Status:** Incomplete

Complete these items for the guarded real-tier full-cycle run (plan 05-09's
`TestAPIKeyFullCycle`) to execute. Everything automatable is already landed
(the guarded test, the responder wiring, the quota ledger row); what remains
is the human-held credential + the unlocked login keyring on the machine with
a real ZCode desktop install. The run itself is the end-of-phase verification
act (orchestrator-run, quota-ledgered — never CI, never an executor step).

## Environment Variables

| Status | Variable | Source | Add to |
|--------|----------|--------|--------|
| [ ] | `ZCODE_ACP_API_KEY` | Local ZCode desktop config — personal zai-api layer (`~/.zcode/v2/` credentials store). Export into the manual-run shell ONLY. Never commit, never write into repo files. | manual-run shell (per-session export) |
| [ ] | `ZCODE_ACP_RAW_CAPTURE` | Your choice — a scratch path OUTSIDE this repository (Phase-1 ground rule 1: raw captures are born outside the repo; the scrubber is the only path into `experiments/captures/`) | manual-run shell (per-session export) |

Optional overrides (defaults are the corpus-locked shapes):

| Status | Variable | Default | Purpose |
|--------|----------|---------|---------|
| [ ] | `ZCODE_ACP_APP_SERVER` | production discovery matrix | Install-root/binary override seam |
| [ ] | `ZCODE_ACP_MODEL` / `ZCODE_ACP_REASONING_LEVEL` | `zai-api/GLM-5.3-Flash` / `high` | Model selection (providerId/modelId) |
| [ ] | `ZCODE_ACP_WORKSPACE` | a test temp dir | Scratch workspace (stable path helps the scrubber's `--redact-path`) |

## Account Setup

- [ ] **Confirm the real z.ai api-key account is active**
  - Skip if: the account behind the local desktop config is already in use
    (the Phase-1 experiments ran on it).

## Dashboard / Session Configuration

- [x] **Unlock the login keyring and confirm the credential entry resolves** (verified 2026-11-20)
  - Location: local GNOME session — gnome-keyring unlocked
  - Verification (field-verified on this machine 2026-11-20 — the username is
    the sha256 of the RESOLVED PATH STRING, not of the file content; the
    key is emitted WITHOUT a trailing newline):
    ```bash
    secret-tool lookup service 'ZCode Credential Encryption' \
      username "$(printf '%s' "$(realpath ~/.zcode/v2/credentials.json)" | sha256sum | cut -d' ' -f1)" | wc -c
    ```
  - Expected: 43 bytes (the 43-char base64url key, no newline). Empty output =
    locked keyring or absent entry — the run's cipher ask then degrades
    honestly (tracked no-answer → turn failure), never a fabricated success.
    (The earlier content-hash form was wrong on both counts — it resolves to
    nothing on this machine; the 05-07 loader's path-string derivation is the
    one the real keyring entry matches.)

## Verification

After exporting the variables (guarded test skips honestly without them):

```bash
# Guard check — SKIPS without ZCODE_ACP_REAL_E2E=1 (the double-guard contract):
go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/
# → "--- SKIP: TestAPIKeyFullCycle" in ~2 ms

# THE REAL RUN (quota-ledgered — see experiments/README.md, Phase-5 row):
export ZCODE_ACP_API_KEY=<key from the local ZCode desktop config>
export ZCODE_ACP_RAW_CAPTURE=<scratch path OUTSIDE this repo>
ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/
```

Expected results:
- PASS — authenticate{zai-api-key} proven by the registry read-back verdict,
  tiny tool-less turn, file-read tool turn, resume → close, staged shutdown,
  zero orphaned processes (`pgrep -f zcode.cjs` empty).
- Then scrub BEFORE any commit:
  ```bash
  node experiments/scrub-capture.mjs "$ZCODE_ACP_RAW_CAPTURE" \
    experiments/captures/EXP-APIKEY-CYCLE-01.jsonl --redact-path <workspace>
  bash experiments/scan-secrets.sh
  ```
- Append the actual spend + verdict line under the PENDING Phase-5 row in
  `experiments/README.md` (the ledger records spend and the V-EXP id
  EXP-APIKEY-CYCLE-01, never credentials).

---

**Once all items complete:** Mark status as "Complete" at top of file.
