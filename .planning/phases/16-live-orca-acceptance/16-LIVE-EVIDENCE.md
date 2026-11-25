# Phase 16 Live Orca Acceptance — Evidence

**Date:** 2026-11-26 (Europe/Moscow)
**Machine:** this Mac (arm64, macOS 27.2)
**Status:** blocked by ZCode account quota, not by the port

## Setup completed

- Built `/Users/nil/bin/zcode-acp` from `cmd/zcode-acp`.
- In Orca Settings → Agents, enabled **OMP**.
- Set the OMP command to `/Users/nil/bin/zcode-acp omp` and committed the field
  (the field reverts unless Enter/blur commits it).
- Orca launched the agent as a structured chat using that command.

## What passed

1. Orca accepted the OMP agent and launched a structured chat
   (`agent: omp`, `kind: chat`, session `a8e682cb-e47d-4e2a-a625-5df1e5b8a2aa`).
2. The bridge log inside the agent run:
   - `app-server resolved layout=mac-app version=4.0.0 root=/Applications/ZCode.app`;
   - `keyring probe class=ok`;
   - `authenticate zai-individual-coding-plan outcome="authenticated (read-back proven)"`.
3. A manual text-task run from the same workspace reached the model call with
   the Darwin provider cache and the real account:
   - `omp --print --mode text` now supplies `reasoningLevel=low` for a
     level-less account selection;
   - the request passed model/level validation.

## Current blocker

Every real generation returns:

```
[1310][Weekly/Monthly Limit Exhausted. Your limit will reset at 2026-11-27 18:05:02]
```

This is the account's ZCode quota, not the wrapper, discovery, PATH, auth, or
Keychain. The Orca chat therefore records `session/prompt: turn ended: failed`
after auth instead of a model reply.

## Resume steps after the reset

1. Confirm `~/bin/zcode-acp omp` is current: `~/bin/zcode-acp --version`.
2. In Orca, open the OMP agent and send:
   `Reply with exactly ZCODE_ORCA_OK and nothing else.`
3. Capture `orca agent read <session-id> --json` with a `message` entry whose
   text is `ZCODE_ORCA_OK`.
4. Record the session id and timestamp here, then mark MORCA-02 complete.

No code change is expected for the resume; only account quota availability.
