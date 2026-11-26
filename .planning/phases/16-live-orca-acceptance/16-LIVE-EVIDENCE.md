# Phase 16 Live Orca Acceptance — Evidence

**Date:** 2026-11-26 03:40 (Europe/Moscow)
**Machine:** this Mac (arm64, macOS 27.2)
**Status:** passed

## Setup

- Built `/Users/nil/bin/zcode-acp` from `cmd/zcode-acp`.
- Orca Settings → Agents: enabled **OMP**.
- OMP command override: `/Users/nil/bin/zcode-acp omp`.
- Selected model in the OMP chat: `ae34a58d-6541-45c8-a83d-a3628b65fc17/DeepSeek-V4-Flash`
  (custom `yadro` OpenAI-compatible provider), effort `max`.

## Root cause found during live testing

The custom model rule in `~/.zcode/v2/provider_config.json` declared
`contextWindow: 262144` but no output-token cap. The backend then requested
247486 completion tokens; with 28850 input tokens the total 276336 exceeded the
model context and the provider returned HTTP 400:

```
Requested token count exceeds the model's maximum context length of 262144 tokens.
You requested a total of 276336 tokens.
```

The model itself was fine: an explicit text run with
`--reasoning-level max` returned `ZCODE_ORCA_OK`.

## Fix applied

Added the missing model option to the custom model rule (local config, not
repo code):

```json
"optionSpecs": { "maxOutputTokens": { "max": 32768 } }
```

The original file is backed up as
`~/.zcode/v2/provider_config.json.bak-zcode-acp-maxout`.

## Live proof

Orca structured chat:

- First passing session: `2db66245-c8cf-4ff6-9dd0-77343fac7c59`.
- Final merged-build session (after integrating upstream Windows support):
  `df489fe3-3dfa-486d-a397-c7722aa45c46`.
- Final Node-backed session: `b23f5213-f9c6-411f-a8d9-a042f0ea2b88`.
- user: `Reply with exactly ZCODE_ORCA_OK and nothing else.`
- assistant: `ZCODE_ORCA_OK`
- state: `done`

Hosted CI for the merged tree (`45732157499`) passed every job: Linux,
macOS arm64, macOS amd64, Windows, Darwin cross-build, govulncheck, secret-scan.

Bridge diagnostics for the same path:

- `layout=mac-app version=4.0.0 root=/Applications/ZCode.app`
- `keyring probe class=ok`
- `session opened`
- prompt finished with `stopReason=end_turn`.

## Dock cleanup

The Electron app-server registers itself and every bundled MCP child as a
foreground LaunchServices app (`type="Foreground"`), producing the Dock's
"ZCode"/"exec" icons. `SpawnArgvForInstall` now launches the same
`Contents/Resources/glm/zcode.cjs` under the host's `node` (login-shell PATH)
on macOS. Verified live: the app-server and all MCP children report
`type="BackgroundOnly"` in `lsappinfo` and the live chat still returns
`ZCODE_ORCA_OK`. The old foreground Electron session was stopped; the
`ZCODE_ACP_DISABLE_NODE_LAUNCHER=1` override keeps the Electron fallback.

## Notes

- The real account's z.ai coding-plan quota is separately exhausted
  (`[1310]`, reset 2026-11-27 18:05:02). The live proof used the custom
  DeepSeek provider, so it does not depend on that quota.
- Orca's conversation-name text task still calls `omp --print --mode text`
  without `--model`; it resolves the desktop default (`zai-api/GLM-5.3`) and
  will keep warning until that default is changed or its quota resets. It does
  not block chat turns.
