<!-- GSD:project-start source:PROJECT.md -->

## Project

**zcode-acp**

Go-обёртка (один статический бинарь), которая выставляет ZCode Desktop наружу как
стандартный **ACP (Agent Client Protocol) v1 сервер поверх stdio**: любой ACP-клиент
может открывать сессии ZCode, отправлять промпты и получать поток ответа
(сообщения, размышления, tool-calls, permission-запросы). Публичная архитектура —
честный ACP-сервер, пригодный для любого клиента. README документирует, среди
прочего, рецепт подключения обёртки к Orca в режиме chat UI.

**Core Value:** ACP-клиент может провести с ZCode полный ход end-to-end — от `initialize` до
стриминга ответа с tool-calls и permission-запросами — на реальном аккаунте z.ai
пользователя (api-key или OAuth coding plan).

### Constraints

- **Tech stack**: Go; единственный бэкенд — desktop app-server ZCode
- **Research policy**: всё про ZCode — сначала исходники
  (https://github.com/zai-org/ZCode), потом эксперименты; всё про Orca —
  сначала исходники (https://github.com/stablyai/orca)
- **Platform**: код кроссплатформенный без ограничений, но v1 поддерживает
  только Ubuntu GNOME (зависимость от Secret Service/gnome-keyring при
  расшифровке кредов); остальные окружения — через PR
- **Docs language**: English (README, рецепты, docs/)
- **Repo config**: по образцу https://github.com/Djarvur/goswitch — строгий
  линтер, GitHub Actions, protected branch
- **Git workflow**: фича-ветки от свежего master/master-аналога, напрямую в
  основную ветку не работаем
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->

## Technology Stack

## The Headline Answer (quality gate)

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go toolchain | 1.27.x (stable 1.27.4) | Language/runtime | Current stable per go.dev/dl (2026-11-19; local machine runs 1.27.3); supported branches are 1.27 and 1.26 only. Single static binary, first-class `os/exec` (`WaitDelay`, `Cancel` for child-process hygiene), `encoding/json`, `context` — everything this adapter needs. Pin in mise.toml, `go 1.27` in go.mod (confidence: HIGH — go.dev/dl) |
| github.com/coder/acp-go-sdk | v0.20.0 | ACP v1 front leg (client-facing) | De facto Go ACP SDK: stdlib-only, both Agent/Client sides, types generated from official ACP schema (**1.25.0**; wire protocolVersion 1 — matches Orca's client), stdio newline-delimited JSON-RPC 2.0 framing included, unsupported optional methods answered `-32601` per spec, extension methods supported (confidence: HIGH — repo README raw + pkg.go.dev + ACP community page, three independent sources) |
| Hand-rolled JSONL codec (stdlib bufio + encoding/json) | ~200 LOC | ZCode app-server back leg | Protocol is JSONL but NOT JSON-RPC (`jsonrpc` key rejected — verified experimentally, HIGH); no library exists or applies. Own the method constants and event types mirroring `packages/zcode-protocol` in zcode-src. Single writer goroutine owns child stdin; correlate by request id; treat `interaction/*` as first-class inbound requests (confidence: HIGH) |
| github.com/godbus/dbus/v5 | v5.2.2 | Secret Service (gnome-keyring) access for the OAuth path | The OAuth coding-plan path must fetch the credential-encryption key from Secret Service (FINDINGS.md: `secret-tool lookup service "ZCode Credential Encryption"`), then AES-256-GCM-decrypt `~/.zcode/v2/credentials.json`. godbus is pure Go (static binary preserved), battle-tested, and the exact library goswitch already uses (confidence: HIGH — FINDINGS + local goswitch go.mod) |
| stdlib crypto/aes + cipher.NewGCM | — | Credential decryption (`enc:v2:os:...` blobs) | AES-256-GCM is in the standard library; no dependency warranted (confidence: HIGH — FINDINGS.md) |
| stdlib net/http + encoding/json | — | z.ai OAuth/PAT minting (getCustomerInfo → api_keys → access_tokens) | Three plain HTTPS calls with exact shapes documented in FINDINGS.md; an SDK would guess wrong. Configure `http.Client` explicitly (IPv6-first and dead-HTTPS_PROXY are FINDINGS landmines) (confidence: HIGH — FINDINGS.md) |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/sebdah/goldie/v2 | v2 | Golden-file transcript tests (fixtures in `testdata/`, `go test -update`, `AssertJson` pretty-diff) | Protocol regression tests layer (below). A ~30-line hand-rolled `-update` helper is the acceptable goswitch-style stdlib alternative (confidence: MEDIUM-HIGH — repo checked 2026-11-19) |
| agentclientprotocol/acp-tck | pinned tag, via `uvx` (Python) | Official external conformance harness: spawns the agent as a stdio subprocess, checks initialize negotiation, session lifecycle, prompt turns, cancellation, error codes, transport hygiene, permissions, capability negatives; MANDATORY/CAPABILITY/ADVISORY tiers → `VERDICT: CONFORMANT`, `--report-json` | Separate **non-blocking** CI job + pre-release manual gate. Upstream self-declares experimental ("may change without notice") — pin the tag, treat verdicts as signal, not a release gate (confidence: MEDIUM-HIGH — official repo, 2026-11-19) |
| github.com/sourcegraph/jsonrpc2 | v0.2.3 | JSON-RPC 2.0 over stdio (client+server, any io.ReadWriteCloser, AsyncHandler) | **Only as fallback** if acp-go-sdk proves unusable — then hand-roll ACP types on top of it. Do NOT pull it alongside the SDK (confidence: MEDIUM — pkg.go.dev) |
| golang.org/x/vuln (govulncheck) | @latest | CI vulnerability gate | Always, in its own pr-sanity job (goswitch pattern) |
| stdlib `flag` | — | CLI surface (`acp` subcommand, `--version`, `--session-dir`) | Manual subcommand dispatch; the CLI must eventually mimic the masked agent's flags exactly (`omp acp --session-dir`) — that surface is tiny, keep it dependency-free (confidence: MEDIUM; exact flags are Phase-1 research) |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| mise | Single source of tool versions (local + CI) | `mise.toml [tools]`: `go = "1.27"`, `golangci-lint = "2.15.0"`, `goreleaser = "2.20.2"`; CI runs `mise install` + `mise run ci` — exact local/CI parity (goswitch D-09 pattern, confidence: HIGH — local). Deliberate tightening over goswitch: pin the linter **exactly** (goswitch floats `"2"`); the strict-config linter set churns (goswitch's disable list already battles `wsl`→`wsl_v5`, `gomodguard`→`gomodguard_v2` deprecations) and only an exact pin reproduces lint results |
| golangci-lint | v2.15.0 (latest, 2026-10-16 — official releases) | Strict config: `version: "2"`, `linters.default: all` with documented disables (wsl/wsl_v5, gomodguard(_v2), noinlineerr, modernize, nonamedreturns), complexity gates (gocyclo 20 / cyclop 15 / gocognit 30), lll 120, tagliatelle snake_case json, gocritic diagnostic+performance+style, minimal revive set, depguard denying unsafe, `generated: strict` + `common-false-positives` exclusions, per-test-file relaxations — copy goswitch `.golangci.yml` and change only `goimports.local-prefixes` + project-specific exclusion comments (confidence: HIGH — local) |
| goreleaser | v2.20.2 (latest stable, 2026-11-12) | Release on `v*` tags: single static binary, checksums, GitHub Releases; `fetch-depth: 0`; `permissions: contents: write` confined to release.yml (goswitch pattern; goswitch itself pins 2.18.1) |
| govulncheck | golang.org/x/vuln | Separate CI job (`go run ...@latest`); reachable-vulnerability findings fail the build; not part of the local gate |
| GitHub Actions | pr-sanity.yml / release.yml / e2e-matrix.yml / security-scheduled.yml | Copy goswitch shape: mise-based gates (build / vet / lint / test -race / tidy-diff) + separate govulncheck job; `permissions: contents: read` everywhere except release; concurrency groups (cancel for PRs, serial queue for release and desktop-e2e); `actions/checkout@v4` as goswitch pins (v5–v7 exist; v5+ = node24 runtime, runner ≥ 2.327.1 — bump consciously or mirror) |
| Self-hosted runner | Orca desktop E2E only | Live GNOME session cannot run on hosted CI; goswitch's e2e-matrix pattern — workflow_dispatch-only, serial concurrency, `contents: read` — is the proven template (confidence: HIGH — local) |

## Installation

# Project bootstrap

# Runtime dependencies (coder SDK is the only must; godbus for the OAuth keyring path)

# Test-time (optional)

# Toolchain via mise (single source of versions, mirrored by CI)

# Gates (mise tasks, goswitch style)

## Testing Patterns (stdio child-process programs)

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| coder/acp-go-sdk v0.20.0 (schema 1.25.0) | Hand-rolled ACP types on sourcegraph/jsonrpc2 v0.2.3 | If a schema bump in the SDK lags a wire change Orca needs, or v0.x churn bites twice in one milestone — but only behind the same `internal/acp` port so the swap is one package |
| coder/acp-go-sdk | ironpark/acp-go (schema 1.32.0 + draft v2 router, encoding/json/v2, Go 1.27+, 36 stars, no releases) | If we later need draft-ACP-v2 or newest schema additions; today it is unproven next to coder's adoption |
| coder/acp-go-sdk | BrokkAi/acp-go (schema v1.29.1, 2 stars), eino-contrib/acp, spachava753/acp-sdk, Tangerg/acp, caelis-labs/acp-go-sdk | Never as first choice — fewer users, thinner maintenance; eino-contrib/acp is the only one worth re-evaluating if coder's stalls (confidence: MEDIUM — community page) |
| godbus/dbus/v5 direct | github.com/zalando/go-keyring | Only if Secret Service lookups grow beyond the single credential-key query — go-keyring is a thin godbus wrapper; direct godbus matches goswitch house style and exposes the exact service/username lookup we need |
| mise.toml single-source CI | actions/setup-go + golangci/golangci-lint-action (ecosystem default) | If the repo ever drops mise — goswitch rejected the action on purpose: its `version:` input would pin the linter a second time and drift from mise.toml (PROJECT.md mandates the goswitch scaffold, so mise wins) |
| stdlib `flag` subcommands | spf13/cobra | Only if the CLI surface (agent-mimicry flags) grows past ~5 subcommands — cobra's weight is unjustified for a bridge binary |
| goldie/v2 golden files | hand-rolled `-update` helper | goswitch-strict stdlib purism; goldie only saves ~30 lines |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Any JSON-RPC library on the ZCode backend leg (sourcegraph/jsonrpc2, jrpc2, net/rpc) | The app-server rejects the `jsonrpc` key outright — framing is plain JSONL with its own namespace; JSON-RPC wrapping corrupts every request (verified experimentally, HIGH) | Hand-rolled bufio+encoding/json codec |
| stdlib net/rpc even where JSON-RPC *is* spoken | It implements the JSON-RPC 1.0-style wire format — no notifications, no 2.0 semantics | coder SDK's Connection (front leg) |
| gojsonrpc, filecoin-project/go-jsonrpc, go.lsp.dev/jsonrpc2 | Stale, niche, or LSP-framing-oriented; no fit | coder/acp-go-sdk (front leg) |
| LSP-style framing (Content-Length headers) | Neither ACP nor ZCode uses it — both are newline-delimited JSON | Newline JSONL framing (SDK does ACP side automatically) |
| `bufio.Scanner` defaults on the ZCode leg | 64 KiB default line limit truncates large `state.updated`/context payloads | Set `Scanner.Buffer` explicitly (tens of MiB) or use `bufio.Reader` |
| Anything writing to stdout except the ACP stream | One stray `fmt.Println` corrupts the protocol stream and desyncs the client | `log/slog` to stderr and/or a file; app-server logs read from its stderr |
| creack/pty and terminal libraries | ACP is a pipe protocol; Orca itself runs agents with `PI_NO_PTY=1` — no PTY anywhere in v1 | exec.Cmd with os.Pipe |
| ORM/config frameworks, DI containers | A bridge binary has ~3 config inputs | Plain structs + stdlib flag |
| ZCode CLI (`zcode` headless CLI) as backend | Explicitly out of scope in v1 (PROJECT.md) | Desktop app-server only |
| Waiting for / building toward an official Go ACP SDK | None announced; official org ships TS/Rust/Python/Kotlin/Java only | coder/acp-go-sdk behind `internal/acp` port |

## Stack Patterns by Variant

- Implement `acp.Agent`; wire `acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)`; await `<-conn.Done()`.
- Map inbound: `initialize` → spawn/adopt app-server + capability report; `session/new` → `session/create` (+ `session/setModel` afterwards — `session/create` drops `options.reasoningLevel` for thinking models, HIGH-confidence FINDINGS pitfall); `session/prompt` → `session/send`; `session/cancel` → app-server cancellation.
- Map outbound: app-server message deltas → `SessionUpdate` (agent_message_chunk / agent_thought_chunk); tool events → tool_call updates; `interaction/requestPermission` → `RequestPermission` RPC back to the ACP client.
- Own codec: `internal/zcode` with typed `Request`/`Response`/`Event` envelopes, method constants as seen in zcode-src `packages/zcode-protocol`.
- Host-managed obligations are non-negotiable: the wrapper MUST answer every `interaction/*` (requestCredentialCipher, requestProviderRuntimeHeaders, requestPermission) — unanswered interactions stall the app-server (HIGH confidence, FINDINGS). `headersApplied:false` for account providers = guaranteed turn failure; answer `{headersApplied:true, requestAuth:{apiKey,apiKeyId}}` with the minted PAT.
- OAuth snapshot revision is the exact string `zcode-builtin:<rev>:<sha256(builtin path)>` — never a bare number; PAT minting per FINDINGS 3.3.
- Process discovery/spawn: `ELECTRON_RUN_AS_NODE=1 <app> resources/glm/zcode.cjs app-server --stdio`.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| go 1.27.x | acp-go-sdk v0.20.0 (requires ≥ go 1.21) | No conflict; stdlib-only SDK, zero transitive deps |
| acp-go-sdk v0.20.0 | ACP wire protocol version 1, schema pin **1.25.0** (`acp.SchemaVersion`) | Matches Orca's ACP v1 client (`protocolVersion: 1` in initialize — FINDINGS). Schema drift is additive; upgrade path = regenerate via `cmd/generate` |
| godbus/dbus/v5 v5.2.2 | go 1.27, pure Go, CGO_ENABLED=0 | Static binary; works on Ubuntu GNOME (Secret Service) out of the box |
| golangci-lint v2.15.0 | go 1.27 projects | v2 config format (`version: "2"`) required — goswitch config is already v2 |
| goreleaser v2.20.2 | single Go binary, GitHub Releases | Pin in mise.toml exactly (goswitch pins 2.18.1; either works — pin one) |
| acp-tck pinned tag | agents speaking protocol version 1 (`--protocol-version 1`) | Experimental upstream — pin tag, non-blocking job |

## Sources

- Local (HIGH): Djarvur/goswitch (v1.2.3, `main`@133a1d6) — `.golangci.yml`, `mise.toml`, `.github/workflows/{pr-sanity,release,e2e-matrix,security-scheduled}.yml`, `go.mod`, stdlib-only test style (scaffold model mandated by PROJECT.md)
- Local (HIGH): FINDINGS.md (private research notes, outside this repo) — ZCode app-server JSONL protocol facts, OAuth/PAT flows, Orca ACP v1 client behavior (experimentally verified 2026-11-18/19)
- Local (HIGH): ZCode source tree (github.com/zai-org/ZCode; local clone) — packages/zcode-protocol method shapes
- github.com/agentclientprotocol org — all 15 repos enumerated: official SDKs TS/Rust/Python/Kotlin/Java, no Go; includes `acp-tck` (HIGH, 2026-11-19)
- github.com/coder/acp-go-sdk (README raw source + repo page) and pkg.go.dev/github.com/coder/acp-go-sdk — v0.20.0 (2026-11-18), **schema pin 1.25.0** quoted verbatim, `ProtocolVersionNumber=1`, both sides, stdio framing, zero deps, 18 importers (HIGH — three independent looks, 2026-11-19)
- agentclientprotocol.com — intro (JSON-RPC over stdio), `/libraries/community.md` (7 Go community libs), `/llms.txt` index (HIGH, 2026-11-19)
- github.com/ironpark/acp-go (schema 1.32.0, draft v2, 36 stars) and github.com/BrokkAi/acp-go (schema v1.29.1, 2 stars) — competitor survey (MEDIUM-HIGH, 2026-11-19)
- github.com/agentclientprotocol/acp-tck — TCK usage, tiers, verdicts, experimental status (MEDIUM-HIGH, 2026-11-19)
- go.dev/dl — Go 1.27.4 stable, 1.27/1.26 supported (HIGH, 2026-11-19)
- github.com/golangci/golangci-lint/releases — v2.15.0 (2026-10-16) (HIGH, 2026-11-19)
- github.com/goreleaser/goreleaser/releases — v2.20.2 (2026-11-12) (HIGH, 2026-11-19)
- github.com/actions/checkout — v4 maintained; v5–v7 exist, v5+ = node24 runtime (MEDIUM, 2026-11-19)
- github.com/sourcegraph/jsonrpc2 — survey only (MEDIUM, 2026-11-19)
- github.com/sebdah/goldie — v2 usage, `-update` (MEDIUM-HIGH, 2026-11-19)
- pkg.go.dev/os/exec — confirms TestHelperProcess is a stdlib-test convention, not documented API (MEDIUM, 2026-11-19)

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-fast` for a trivial task inline, with no subagents and no PLAN.md
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
