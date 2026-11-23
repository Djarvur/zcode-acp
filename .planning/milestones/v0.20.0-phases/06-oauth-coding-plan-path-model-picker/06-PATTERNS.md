# Phase 6: OAuth Coding-Plan Path & Model Picker - Pattern Map

**Mapped:** 2026-11-22
**Files analyzed:** 13 (5 new, 8 modified — plus the conditional zcode method constant and fake-corpus scenarios)
**Analogs found:** 13 / 13 (all new files have in-repo analogs; every modified file IS its own analog)

**Tracked-source note:** every analog named below was verified with `git ls-files` (17/17 tracked). The external reference clones (`/tmp/zcode-acp-ref-study/repo` v0.89.0, `~/DiskD/W/Djarvur/zcode-src` 3.17.1) are cite-and-point ONLY — their byte-exact wire shapes live in 06-RESEARCH.md Patterns 1–8 and must never be treated as in-repo analogs.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/interact/credstore.go` (NEW) | service (credential reader) | file-I/O + decrypt | `internal/interact/cipherdecrypt.go` | exact (same package, same seam family) |
| `internal/hostsvc/patminter.go` (NEW) | service (HTTPS client + cache) | request-response (network) | `internal/hostsvc/snapshot.go` (KeySource/cache posture) + `internal/interact/cipherdecrypt.go` (coded errors, bounded exec) | role-match (HTTPS-client half has no in-repo analog — use RESEARCH Pattern 3 wire shapes) |
| `internal/acpfront/configoptions.go` (NEW) | service (translator) | transform | `internal/hostsvc/readback.go` (dig-out helpers) + `internal/acpport/agent.go` (neutral-vocabulary translation) | exact (composition of two proven patterns) |
| `test/realdesktop/codingplan_test.go` (NEW) | test (guarded real tier) | request-response E2E | `test/realdesktop/apicycle_test.go` | exact (sibling file) |
| `internal/zcode/fake/testdata/scenarios/coding-plan-auth-cycle.json` + `coding-plan-probe-fail.json` + `runtime-headers-pat.json` + config-option round-trips (+ `.golden`) (NEW) | test fixture (fake corpus) | event-driven scripted | `internal/zcode/fake/testdata/scenarios/{auth-apikey,runtime-headers-answer,credential-cipher-scripted}.json` | exact |
| `internal/interact/cipherdecrypt.go` (MOD) | service (decrypt) | transform | itself (`SecretToolKeyLoader` gains the v1-fallback sibling loader) | self |
| `internal/interact/responder.go` (MOD) | middleware (responder policy) | event-driven (reverse requests) | itself (`handleRuntimeHeaders` + `RuntimeHeadersKeySource` seam widen to mode-aware/accountScope) | self |
| `internal/hostsvc/snapshot.go` (MOD) | service (snapshot builder) | transform | itself (`BuildRevision` gains the account-overlay variant; empty api-key overlay untouched) | self |
| `internal/acpfront/auth.go` (MOD) | controller (authenticate arm) | request-response | itself (`Authenticate` dispatch + `authenticateAPIKey` chain — the coding-plan arm clones the spine) | self |
| `internal/acpfront/front.go` (MOD) | controller (NewSession/set_config_option) | request-response | itself (`NewSession` create→subscribe→setModel) | self |
| `internal/acpport/agent.go` (MOD) | adapter (wire translation) | request-response | itself (`Authenticate` delegation replaces `SetSessionConfigOption`'s -32601; `initializeResponse` gains the options advertisement) | self |
| `internal/acpport/types.go` (MOD) | types (neutral vocabulary) | transform | itself (`Handlers` interface + `AuthMethodDef` pattern for the new `ConfigOption` DTOs) | self |
| `cmd/zcode-acp/main.go` (MOD) | config/wiring | — | itself (`buildFront` + `wireResponder` gain the PAT cache and mode-aware source; NO new user flags per D-12) | self |
| `internal/zcode/envelope.go` (MOD, conditional on OQ1) | model (method constants) | — | itself (constants block lines 140–154) | self |

## Pattern Assignments

### `internal/acpfront/auth.go` — the coding-plan authenticate arm (controller, request-response)

**Analog:** itself — `authenticateAPIKey` (lines 125–178) is the structural spine the new arm clones; the `Authenticate` dispatch (lines 88–117) is where the Phase-6 decline arm is replaced.

**The dispatch being replaced** (lines 101–108 — the honest-decline arm that becomes the real chain):
```go
if methodID != AuthMethodZaiAPIKey {
    // Honest disposition for the advertised-but-Phase-6 method: never a
    // fake success, never the misleading -32601 default (D-02).
    authLog(methodID, "declined (Phase 6)")

    return acpport.NewInternal(
        "authenticate %s: the OAuth coding-plan flow lands in Phase 6", methodID)
}
```

**The chain to clone** (lines 137–177 — watcher BEFORE push, build, push, refresh, probe gate, Await verdict; the coding-plan arm inserts the keyring probe FIRST and the PAT mint LAST):
```go
watchOpts := []hostsvc.WatchOption{}
if f.auth.ReadbackDeadline > 0 {
    watchOpts = append(watchOpts, hostsvc.WithReadbackDeadline(f.auth.ReadbackDeadline))
}

watcher, remove := hostsvc.WatchRegistry(f.conn, f.auth.WantModels, watchOpts...)
defer remove()

snap, err := hostsvc.NewSnapshotBuilder(f.auth.Keys).Build(f.auth.Install)
if err != nil {
    return acpport.NewInternal("authenticate: build snapshot: %v", err)
}

if _, err := hostsvc.PushEnvelope(ctx, f.conn, snap.Params); err != nil {
    return acpport.NewInternal("authenticate: push snapshot: %v", err)
}

if err := hostsvc.RefreshRegistry(ctx, f.conn); err != nil {
    return acpport.NewInternal("authenticate: refresh registry: %v", err)
}

if !watcher.Applied() {
    f.probeCreateReadback(ctx, watcher)
}

if _, err := watcher.Await(ctx); err != nil {
    return acpport.NewInternal("authenticate: registry read-back: %v", err)
}
```

**Error-handling pattern** (every step wraps into a coded error naming the step; transport noise never passes raw — line 124 doc + each return). Coded-error constructors live in `acpport/types.go:288-303` (`NewInvalidParams` -32602, `NewInternal` -32603).

**Composition sketch for the new arm** is prescriptive in 06-RESEARCH.md lines 903–927 (`authenticateCodingPlan`): probe (D-02) → `f.auth.Creds.LoginToken(ctx, "zai")` (D-03) → identical read-back setup → `BuildAccount` overlay (Pattern 2) → push → refresh → probe gate → Await → `f.auth.Pat.Ensure(ctx, login)` (D-04) → record auth mode (`f.authMode.Store(authModeCodingPlan)`, the D-11 seam the runtime-headers source reads). Extend `AuthConfig` (lines 55–74) with `Creds`, `Pat`, and `Probe` seams instead of new globals.

---

### `internal/interact/cipherdecrypt.go` — the v1 fallback loader + store reader seed (service, transform)

**Analog:** itself — the v2 loader family is the template the enc:v1 sibling joins.

**Loader construction contract** (lines 341–357 — `LookupCommand` returns `(argv, env, err)`; tests inspect the contract without executing; the v1 key derivation follows the same shape):
```go
func (l *SecretToolKeyLoader) LookupCommand() (argv, env []string, err error) {
    if l.bin == "" || l.credPath == "" {
        return nil, nil, errEmptyCredentialsPath
    }

    link, err := resolveRealPath(l.credPath)
    if err != nil {
        return nil, nil, err
    }

    sum := sha256.Sum256([]byte(link))
    username := hex.EncodeToString(sum[:])

    argv = []string{l.bin, "lookup", "service", secretToolService, "username", username}

    return argv, l.env, nil
}
```

**AES-256-GCM core** (lines 219–243 — `decryptOne`'s open; the v1 fallback decrypts with the same primitive, different key + segment layout `enc:v1:<iv>.<tag>.<ct>`, key = `sha256(secret)`):
```go
block, err := aes.NewCipher(key)
...
gcm, err := cipher.NewGCM(block)
...
sealed := make([]byte, 0, len(ct)+len(tag))
sealed = append(sealed, ct...)
sealed = append(sealed, tag...)

plaintext, err := gcm.Open(nil, iv, sealed, nil)
```

**Coded decline taxonomy** (lines 61–67) — copy the shape for credstore entry errors:
```go
const (
    CipherCodeUnsupportedOperation = "unsupported-operation"
    CipherCodeInvalidValue         = "invalid-value"
    CipherCodeKeyUnavailable       = "key-unavailable"
    CipherCodeKeyMismatch          = "key-mismatch"
    CipherCodeDecryptFailed        = "decrypt-failed"
)
```

**Layout-parse discipline** (lines 249–279 `parseEncV2OS`): `strings.Cut`/`strings.Split` with exact segment counts, base64url decode (`base64.RawURLEncoding`), every malformed shape a coded failure — never a panic, never plaintext passthrough of unknown `enc:` prefixes (RESEARCH Pattern 1's `hasUnhandledEncPrefix` rule).

**Hygiene posture** (file doc lines 34–41, test-locked): plaintext exists only in reply-construction scope; logs carry shapes/lengths (`"plaintext_len", len(value)`, line 166) never values; subprocess env via `ScrubbedEnv` (lines 426–444, allowlist `PATH, HOME, DBUS_SESSION_BUS_ADDRESS, XDG_RUNTIME_DIR`). The credstore reader inherits this verbatim.

---

### `internal/interact/credstore.go` (NEW) — credentials.json reader (service, file-I/O + decrypt)

**Analog:** `internal/interact/cipherdecrypt.go` (same package; JSON decode posture, per-entry coded failure isolation, hygiene).

**Decode posture to copy** — `Handle`'s params decode (lines 139–152): decode into a minimal anonymous struct, undecodable input is a coded failure not an error. The store reader inverts it: one undecryptable entry isolates to that entry (RESEARCH Pattern 1: per-entry error isolation; unreadable store → empty map, never a panic).

**KeyLoader seam extension** (lines 112–119):
```go
type KeyLoader interface {
    LoadKey(ctx context.Context) ([]byte, error)
}
```
The v1 fallback loader implements the SAME interface (D-01: the seam is the upgrade boundary). `LoadKey`'s exec discipline (lines 383–404): `exec.CommandContext` (bounded ctx — AUTH-04), `cmd.Output()` (stdout captured/drained, stderr rides the wrapped error, never logged), wrapped sentinels (lines 98–102: `errLoaderLookup`, `errLoaderKeyShape` — sentinels wrapped with `%w` at the failure site, never constructed dynamically, err113).

**Entry keys the reader serves** (values never logged): `oauth:<family>:access_token` (strip `Bearer ` prefix), `zcodejwttoken`, `project-key-location:v1:…` — shapes in RESEARCH Pattern 1.

---

### `internal/interact/responder.go` — mode-aware runtime-headers source (middleware, event-driven)

**Analog:** itself.

**The seam to widen** (lines 90–94 — RESEARCH Pattern 4/A9: widen so the coding-plan source can return accountScope; keep the api-key two-field answer byte-compatible):
```go
type RuntimeHeadersKeySource func(ctx context.Context, req Request) (apiKey, apiKeyID string, err error)
```

**The success-arm builder to extend** (lines 421–449 — the accountScope key joins `requestAuth` only when the mode supplies it):
```go
func (r *Responder) handleRuntimeHeaders(ctx context.Context, req Request) (Answer, error) {
    if r.keySource == nil {
        return Answer{Err: &zcode.WireError{
            Code:    zcode.CodeInternalError,
            Message: "interaction/requestProviderRuntimeHeaders unavailable: no api-key source configured",
        }}, nil
    }

    apiKey, apiKeyID, keyErr := r.keySource(ctx, req)
    if keyErr != nil {
        // A bare handler error wraps as the coded -32603 internal error at
        // the deliver seam — the honest coded-ERROR arm, never
        // headersApplied:false (PITFALL 3).
        return Answer{}, fmt.Errorf("interaction/requestProviderRuntimeHeaders: %w", keyErr)
    }

    auth := map[string]any{"apiKey": apiKey}

    if apiKeyID != "" {
        auth["apiKeyId"] = apiKeyID
    }

    return Answer{Result: map[string]any{
        "headersApplied": true,
        "requestAuth":    auth,
    }}, nil
}
```
Never `headersApplied:false` (PITFALL 3, lines 415–420 doc); mint failure = the coded ERROR envelope.

**Seam-injection pattern** (lines 137–150 `WithCredentialDecrypt` — the model for any new mode-aware option):
```go
func WithCredentialDecrypt(svc *CredentialDecrypt) Option {
    return func(r *Responder) {
        r.cipher = svc
        r.methods[zcode.MethodRequestCredentialCipher] = MethodPolicy{
            Deadline: DefaultCredentialCipherDeadline,
            Handler:  svc.Handle,
        }
    }
}
```
Policy-table row for runtime-headers: lines 282–285 (`Deadline: DefaultRuntimeHeadersDeadline` = 180 s, line 48).

**D-05 re-mint trigger inputs** arrive on the ask params: `expectedAccountScope` / `rejectedProjectTokenFingerprint` (64-hex). Dig them the way `normalize`/`paramsString` do (lines 492–517) — camelCase top-level string dig, undecodable params classify as absent. Nested/typed dig-out: `json.Unmarshal` into a params struct like `CredentialDecrypt.Handle` line 139.

---

### `internal/hostsvc/snapshot.go` — the account-provider overlay builder (service, transform)

**Analog:** itself — `BuildRevision` assembles the strict envelope; the extension adds the NON-empty providers/states variant (api-key empty overlay untouched).

**The envelope assembly to extend** (lines 134–146):
```go
func (b *SnapshotBuilder) BuildRevision(install supervisor.Install, rev int) Snapshot {
    basedOn := supervisor.BuiltinRevision(install.BuiltinFile, rev)
    revision := b.revision()

    params := map[string]any{
        "revision":                    revision,
        "basedOnZCodeBuiltinRevision": basedOn,
        "providers":                   map[string]any{},
        "states":                      map[string]any{},
    }

    return Snapshot{Revision: revision, BasedOn: basedOn, Params: params}
}
```
New `BuildAccount` (RESEARCH Pattern 2): same formula, but `providers` carries one entry per builtin-table rule with `config.access.type == "zhipu-account"` (read from the SAME `install.BuiltinFile` `readBuiltinRevision` already opens, lines 150–165 — extend to also parse `config.providerConfigRules.providerRules[]`), and `states` carries `{availability:"available", entitled:true, current:true}`. Overlay content mirrors the desktop's own table verbatim (D-08's snapshot-side twin). Entitlement inputs (setting.json selections/modes + store `oauth:<family>:access_token`) come from the credstore reader — read-only toward the desktop, values never logged.

**In-memory-only KeySource posture** (lines 45–68 — the PAT cache struct copies this discipline):
```go
type KeySource interface {
    APIKey(ctx context.Context) (apiKey, apiKeyID string, ok bool)
}
```

**Call + wrap-on-error** (lines 191–198 `PushEnvelope`) and **ack dig-out** (lines 203–228 `parseAck`/`ackString`/`ackInt` — capture-verbatim camelCase map dig, the tagliatelle contract). `RefreshRegistry` (lines 233–241) reused verbatim.

**RuntimeHeadersHandler** (lines 258–267) is the existing api-key source adapter — the coding-plan source adapter (PAT cache → token+id+accountScope) is its sibling.

---

### `internal/hostsvc/patminter.go` (NEW) — PAT mint chain + cache (service, request-response HTTPS)

**Analog:** `internal/hostsvc/snapshot.go` for the in-memory cache posture (`StaticKey`, T-5-15 doc lines 41–44) and coded-error sentinels (lines 247–248 `errAPIKeyUnconfigured` naming the fix); `internal/interact/cipherdecrypt.go` for bounded-context + hygiene discipline. **The HTTPS client itself has NO in-repo analog** — the wire shapes are pinned byte-exactly in 06-RESEARCH.md Pattern 3 + its Go code examples (lines 826–861: raw-token getCustomerInfo, Bearer api_keys list/create, Bearer access_tokens with `{"clientType":"zcode","clientVersion":...}`, and the accountScope sha256 snippet). Explicit `http.Client{Timeout: 15 * time.Second}` transport at construction (RESEARCH Pitfall 7: log effective proxy once, default Happy-Eyeballs dialer, never ProxyFromEnvironment surprises).

**Cache + reactive refresh (D-05)** — port the reference semantics per RESEARCH Pattern 3: cache `{token, apiKeyId, fingerprint, expiresAt, refreshAt, accountScope}` in-process; `rejectedProjectTokenFingerprint` match → exactly one silent re-mint; scope mismatch → coded `project_token_scope_invalidated`-class error; `code:404`/HTTP-404 on access_tokens → drop cached location only (permission/network errors never re-create — Pitfall 8). Success codes {200, 0, "200", "0"}; every business failure a coded stable error. Single-flight concurrent mints (mutex + shared in-flight call).

**Multi-project pick (D-06):** deterministic first personal org/project (`默认机构`/`默认项目` preference, `projectType != "2"`), slog warn listing ids when several — never block.

**Hygiene (Pitfall 4, test-locked):** the cache struct carries no `String()`/log rendering; mint errors log codes and shapes/lengths, never bodies (`resp.Body` never logged). PAT never persisted (D-04) — no file writes anywhere in the package.

---

### `internal/acpfront/configoptions.go` (NEW) — settings → configOptions translator (service, transform)

**Analog A: `internal/hostsvc/readback.go` dig-out helpers** — the exact typed-dig discipline to extend to the full settings snapshot (lines 213–253):
```go
func digCreateAvailable(raw json.RawMessage) []string {
    var obj map[string]any
    if err := json.Unmarshal(raw, &obj); err != nil {
        return nil
    }

    settings, _ := obj["settings"].(map[string]any)

    model, _ := settings["model"].(map[string]any)

    available, _ := model["available"].([]any)
    ...
    out = append(out, pid+"/"+mid)
```
Map-based, type-assert with `, _`, capture-verbatim camelCase wire keys, undecodable = absent (never a verdict/never a crash). The translator consumes ONLY `settings` snapshots from create/setModel/setThoughtLevel replies (D-08/D-10 — Pitfall 6: no wrapper-side filtering, renaming, enrichment; currentValue only from `settings.model.current`/`thoughtLevel.current`).

**Analog B: `internal/acpport/agent.go` neutral-vocabulary translation** (lines 204–230 `newSessionParams`/`mcpServerDef` — build neutral DTOs at the tap, SDK types never cross; union-variant switches like `mcpServerDef`'s lines 218–227 are the pattern for select-option assembly). Wire assembly uses SDK `acp.SessionConfigOptionSelect`/`SessionConfigSelectOption` per RESEARCH Pattern 6 (Do-Not-Hand-Roll: the SDK owns `type:"select"` discriminant injection and the required-list Validate).

**Value serialization:** `ref.providerId + "/" + ref.modelId` (the repo's existing read-back serialization — readback.go:245, main.go:194). `strings.Cut` on the first `/` only for parsing; NO `$` composite values (RESEARCH Pattern 7). The honest-currentValue rule (D-10/A7): omit the model entry rather than fabricate a selection when the backend reports no current — planner locks per Open Question 2.

---

### `internal/acpfront/front.go` — NewSession options + set_config_option handler (controller, request-response)

**Analog:** itself.

**The NewSession leg gaining configOptions** (lines 223–297 — validation-first, create, subscribe, the setModel follow-up whose reply NOW feeds the settings snapshot store):
```go
if !filepath.IsAbs(params.Cwd) {
    return "", acpport.NewInvalidParams(
        "session/new: cwd must be absolute (spec session-setup MUST), got %q", params.Cwd)
}
...
if _, err := f.conn.Call(ctx, zcode.MethodSessionSetModel, map[string]any{
    wireKeySessionID:             sid,
    "model":                      model,
    "persistAsWorkspaceLastUsed": false,
}); err != nil {
    f.closeOrphan(ctx, sid, "setModel", err)
    return "", acpport.NewInternal("session/setModel: %v", err)
}
```
Carry the pattern forward: validate params first (the set_config_option analog of the cwd gate = unknown configId/value-not-listed → `acpport.NewInvalidParams` naming the configId and the allowed set — CONTEXT discretion, spec MUST), then the backend call, then build the response from the REPLY's settings (D-10). The options-omit-shape discipline (lines 259–272: `options` rides ONLY when a level exists) applies to every setModel the picker issues — a model switch carries the CURRENT confirmed level unless the ask changes it (Pitfall 2).

**Reply dig-out pattern** (lines 632–652 `createSessionID`): two-stage unmarshal (result → session map), empty dig = coded internal error naming the method. The settings-snapshot dig for configOptions extends this family; store the latest confirmed settings per session id in a mutex-guarded map bounded with the registry (`recordCwd`/`dropCwd`, lines 505–519 — the WR-07 discipline).

**Front wiring pattern:** new capability = new `Option` (lines 131–157) + struct field; the per-session confirmed-settings store follows `cwds`'s shape.

---

### `internal/acpport/agent.go` — set_config_option delegation + capability advertisement (adapter, request-response)

**Analog:** itself.

**The -32601 default being replaced** (lines 388–394):
```go
func (a *agentAdapter) SetSessionConfigOption(
    ctx context.Context,
    params acp.SetSessionConfigOptionRequest,
) (acp.SetSessionConfigOptionResponse, error) {
    return acp.SetSessionConfigOptionResponse{}, unadvertised(acp.AgentMethodSessionSetConfigOption)
}
```
becomes an `Authenticate`-style delegation (lines 323–332 — validate-in-front, delegate via Handlers, code errors via `wireError`, success = the response assembled from the neutral vocabulary):
```go
if err := a.h.Authenticate(ctx, string(params.MethodId)); err != nil {
    return acp.AuthenticateResponse{}, wireError(err)
}

return acp.AuthenticateResponse{}, nil
```
The response MUST carry the full re-emitted `ConfigOptions` list (SDK Validate requires non-nil — RESEARCH Pattern 7).

**Capability advertisement** (lines 165–188 `initializeResponse` — extend the assembled response with `SessionCapabilities: {Options: &acp.SessionOptionsCapabilities{}}`; the empty-struct opt-in per RESEARCH Pattern 6):
```go
return acp.InitializeResponse{
    ProtocolVersion: acp.ProtocolVersionNumber,
    AgentInfo:       &acp.Implementation{...},
    AuthMethods:     authMethods,
}
```

**wireError discipline** (lines 440–451 — stays unwrapped; the SDK type-asserts `*RequestError`):
```go
var seamErr *Error
if errors.As(err, &seamErr) {
    return acp.NewRequestError(seamErr.Code, seamErr.Message, nil)
}

return acp.NewInternalError(err.Error())
```

---

### `internal/acpport/types.go` — neutral ConfigOption vocabulary (types, transform)

**Analog:** itself — `Handlers` interface extension pattern (lines 23–66, one method + doc contract each) and `AuthMethodDef` (lines 80–84) as the DTO shape for `ConfigOptionDef`/`SelectOptionDef`. Error constructors (lines 288–303) reused for the -32602 invalid-config-option arm:
```go
func NewInvalidParams(format string, args ...any) *Error {
    return &Error{Code: CodeInvalidParams, Message: fmt.Sprintf(format, args...)}
}
```
New Handlers methods: `NewSession` return widens to carry configOptions (or a separate query), and `SetSessionConfigOption(ctx, sessionID, configID, value) (fullList, error)` joins the interface — with the doc-contract comment style of `Authenticate` (lines 59–65).

---

### `cmd/zcode-acp/main.go` — wiring (config, no new user flags)

**Analog:** itself.

**wireResponder gains the mode-aware source** (lines 244–253 — the current seam set; the PAT cache + coding-plan source join as additional options):
```go
responder := interact.NewResponder(
    interact.WithPermissionForwarder(interact.NewPermissionBridge(front.PermissionForward()).Forward),
    interact.WithRuntimeHeadersKeySource(hostsvc.RuntimeHeadersHandler(keys)),
    interact.WithCredentialDecrypt(interact.NewCredentialDecrypt(
        interact.NewSecretToolKeyLoader(secretToolBin, credentialsPath(), interact.ScrubbedEnv(os.Environ())))),
)
```

**buildFront gains the coding-plan seams** (lines 216–231 — `AuthConfig` extensions ride here: Creds/Pat/Probe). `credentialsPath` (lines 202–209) is the existing `~/.zcode/v2/credentials.json` resolver the credstore reader reuses. `parseConfig`'s model parsing (lines 151–154: `strings.Cut(selection, "/")`) is the exact value-split the config-option value parser mirrors. NO new flags/env vars (D-12: desktop login IS the credential).

---

### `test/realdesktop/codingplan_test.go` (NEW) — guarded real-tier sibling (test)

**Analog:** `test/realdesktop/apicycle_test.go` — copy the discipline verbatim:
- Build tag `//go:build realdesktop` (line 1) + runtime guard `ZCODE_ACP_REAL_E2E` (line 78) + skip-when-unconfigured (line 81 doc).
- File-header contract: canonical run command, quota note, evidence/scrubber discipline (lines 11–44) — coding-plan version: NO exported key; the credential is the desktop's own keyring login (D-12); capture id e.g. `EXP-CODINGPLAN-CYCLE-01`; ledger row in `experiments/README.md` (D-13).
- Knob constants block (lines 76–138): env names, deadline (`apiCycleDeadline = 12 * time.Minute`), shutdown watchdog, zero-orphan proof knobs.
- Harness shape: discover → spawn → conn → buildFront/wireResponder → authenticate → session/new → ONE tiny tool-less turn (D-13) → staged shutdown.
- No key material in env, logs, or captures; scrubbed capture before any commit.

---

### Fake corpus scenarios (NEW fixtures + goldens)

**Analogs:** `auth-apikey.json` (provider/updateAccountConfig ack → refreshRegistry with `state.updated` patch side-effect → session/create carrying `settings.model.available` → subscribe → setModel — the exact shape the coding-plan cycle extends with `account:zai-individual-coding-plan` providers and the settings `thoughtLevel` block); `runtime-headers-answer.json` lines 48–58 (the reverse ask with `expect_answer` byte-compare — the PAT-arm twin answers `{headersApplied:true, requestAuth:{apiKey, apiKeyId, accountScope}}`, synthetic secret shapes per the corpus injection discipline); `credential-cipher-scripted.json` lines 14–21 (the reverse cipher ask mid-turn — the coding-plan scenario fires it during authenticate). Reverse-ask scripting (`side_effects[].reverse` + `expect_answer` + `reannounce`) and `eof_drain_ms` are the established semantics; a full annotated coding-plan seed is in 06-RESEARCH.md lines 933–973.

---

### `internal/zcode/envelope.go` — method constant (conditional, model)

**Analog:** itself, constants block (lines 140–154). Add `MethodSessionSetThoughtLevel = "session/setThoughtLevel"` ONLY if the planner locks OQ1 on the dedicated method; `MethodSessionSetModel` (line 150) already exists and is the field-proven default (RESEARCH Open Question 1 recommendation: setModel + `options.reasoningLevel` for BOTH switch kinds).

## Shared Patterns

### Coded errors — never a fabricated success, never raw transport noise
**Source:** `internal/acpport/types.go:288-303` (front-facing), `internal/interact/cipherdecrypt.go:61-67,96-102` (decline codes + sentinels), `internal/hostsvc/snapshot.go:247-248` (actionable naming)
**Apply to:** every new file — the authenticate arm, PAT minter (stable `project_token_*`-class codes per RESEARCH Pattern 3), credstore, set_config_option (-32602 naming configId + allowed set), keyring probe (the error NAMES the missing piece: binary → install hint; no bus → "no desktop session"; no entry → "log in via the ZCode desktop first").
```go
var errAPIKeyUnconfigured = errors.New("api key not configured " +
    "(set ZCODE_ACP_API_KEY or pass the api key at authenticate)")
```

### Bounded context on every exec and network call (AUTH-04)
**Source:** `internal/interact/cipherdecrypt.go:383-404` (`exec.CommandContext` + `cmd.Output`), `internal/hostsvc/readback.go:98-128` (poll-with-deadline)
**Apply to:** keyring probe (`exec.LookPath` first, then the loader argv under a bounded ctx ≤ 5 s, D-02), every mint HTTPS call (15 s per-call), the credstore reads. Failure never hangs the chain (Pitfall 5).

### Key-material hygiene (T-5-17 extended to the PAT, D-04)
**Source:** `internal/interact/cipherdecrypt.go:34-41` (file doc), `:166` (shape/length logging), `:426-444` (`ScrubbedEnv`)
**Apply to:** credstore reader, PAT cache/minter, runtime-headers PAT answers. Values in reply-construction scope only; structural logs carry ids/shapes/lengths; no `String()` on secret structs; no disk writes; scrub + scan-secrets gates on evidence commits (D-13).

### Verdict = registry read-back, never the ack (PITFALL 1)
**Source:** `internal/hostsvc/readback.go:65-128` (`WatchRegistry` before traffic → `Applied()` probe gate → `Await`), used in `auth.go:142-177`
**Apply to:** the coding-plan authenticate arm verbatim — same watcher, same probe-create branch, same coded negative.

### Capture-verbatim camelCase dig-outs (tagliatelle)
**Source:** `internal/hostsvc/readback.go:213-253` (`digCreateAvailable`), `internal/hostsvc/snapshot.go:203-228` (`parseAck`), `internal/acpfront/front.go:632-652` (`createSessionID`)
**Apply to:** settings-snapshot dig (model.available entries incl. `reasoning.levels`, `thoughtLevel` block), runtime-headers params (`expectedAccountScope`, `rejectedProjectTokenFingerprint`), credstore entry keys. Map/type-assert dig, undecodable = absent/empty, never a panic.

### Neutral vocabulary at the acpport seam (SDK types never cross)
**Source:** `internal/acpport/types.go:23-66` (Handlers), `internal/acpport/agent.go:204-230` (DTO translation)
**Apply to:** the ConfigOption DTO family and the SetSessionConfigOption delegation — the front implements Handlers only; agent.go translates.

### TDD RED→GREEN with machine evidence; golden corpus byte-locks; guarded real tier
**Source:** `test/realdesktop/apicycle_test.go:1-44`, fake corpus `testdata/scenarios/*.json` + `.golden` byte-lock
**Apply to:** every wave — new fake scenarios land Wave 0 (RESEARCH Wave 0 Gaps), the real-tier test is human-gated and late-phase (D-12).

## No Analog Found

| File | Role | Data Flow | Reason | Planner Guidance |
|------|------|-----------|--------|------------------|
| `internal/hostsvc/patminter.go` HTTPS client half | service | request-response (HTTPS) | No outbound-HTTPS client exists anywhere in the repo (all prior network I/O is stdio pipes + local exec) | Use 06-RESEARCH.md Pattern 3 wire shapes + its Go code examples (lines 826–861) verbatim on stdlib `net/http` with explicit `http.Client{Timeout}`; structure the package (sentinel errors, in-memory cache, single-flight) after `snapshot.go`'s `StaticKey` + `cipherdecrypt.go`'s discipline. STACK.md locks stdlib — an HTTP SDK would guess shapes the reference pins byte-exactly. |

Everything else has an exact or self analog.

## Metadata

**Analog search scope:** `internal/{acpfront,acpport,hostsvc,interact,zcode,supervisor,core,turn}`, `cmd/zcode-acp`, `test/realdesktop`, `internal/zcode/fake/testdata/scenarios`
**Files scanned:** 74 Go files enumerated; 12 read in full (+1 file header read targeted); 3 corpus scenarios read in full
**Analog search stopped:** after the full read of the 8 modified files + their 4 fixture/test siblings — beyond the 3–5-analog floor because most targets ARE their own analogs (extension phase)
**Pattern extraction date:** 2026-11-22
