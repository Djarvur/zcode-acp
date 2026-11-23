# Phase 7: Hardening, VM & Manual E2E, Documentation - Pattern Map

**Mapped:** 2026-11-22
**Files analyzed:** 14 (8 modified, 6 new)
**Analogs found:** 14 / 14 — every new/modified file has an in-repo analog; no "No Analog Found" entries

**Tracked-source gate:** every analog path below was verified with `git ls-files -- <path>` (all printed, all tracked). No gitignored mirrors used.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/zcode-acp/main.go` (mod: omp subcommand + `--model` + version stamp) | CLI entry/config | request-response (argv parse → launch) | itself: flag block :81-114, `parseConfig` :135-164 | exact (extends own parse) |
| `cmd/zcode-acp/main_test.go` (ext: omp parse unit tests) | test | request-response | `TestWiringCodingPlanSeams` :102, `TestAgentBinaryEndToEnd` :120 | exact |
| `internal/acpport/agent.go` (mod: delete arm, fork arm, detail-carrying errors, IN-02) | adapter/controller | request-response (JSON-RPC arm) | itself: `CloseSession` :402-415, `unadvertised` :554-560, `wireError` :541-552 | exact |
| `internal/acpport/types.go` (mod: `Error` gains data/details + `NewInternalWithData`) | types/utility | transform | itself: `Error` :333-364, code constants :325-331 | exact |
| `internal/acpfront/front.go` (mod: `/model` intercept, WR-01, WR-04, IN-04) | service (session policy) | event-driven + request-response | itself: `Prompt` :347-419, `SetSessionConfigOption` :510-577, `setConfigMutation` :590-646, `applyConfiguredModel` :718-758, `closeOrphan` :886-896 | exact |
| `internal/acpfront/lifecycle.go` (mod: WR-05, WR-04 resume arm) | service (lifecycle) | event-driven (teardown) | itself: `ResumeSession` :63-138, `CloseSession` :142-212 | exact |
| `internal/hostsvc/snapshot.go` (mod: IN-01 single-read fix) | service | transform (build push envelope) | itself: `Build` :139-146 (single-read discipline) | exact |
| `internal/interact/keyringprobe.go` (mod: IN-03 classify order/comment) | utility | request-response (subprocess probe) | itself: `classify` :191-241 | exact |
| `internal/acpfront/conformance_test.go` (new: named conformance suite) | test | composed E2E | `fullcycle_test.go` `TestFullCycle` :50-69; `harness_test.go` `ComposeFront` :410-460 | exact |
| SESS-02/03 corpus + conformance rows (new test files/goldens) | test | composed E2E + golden | `golden_test.go` `TestGolden` :365-398 driver discipline; existing acpfront `*_test.go` files | exact |
| `scripts/vm-*.sh` (new: VM scenario) | utility/script | batch (process driver) | `experiments/check-docs.sh` (gate shape); `main.go` `discoverInstall` :463-490 (probe rows) | role-match (first repo script outside experiments/) |
| `README.md` (mod: English rewrite) | documentation | — | itself (section skeleton); `docs/protocol/README.md` (provenance legend) | exact |
| `docs/*.md` new (launch recipe, protocol notes/provenance/limitations) | documentation | — | `docs/protocol/README.md` deliverables table + provenance legend; `experiments/check-docs.sh` gate contract | exact |
| `07-UAT.md` + `experiments/README.md` ledger row (new/mod) | documentation (manual-tier record) | — | `.planning/phases/06-*/06-UAT.md`; `experiments/README.md` ground rules | exact |

## Pattern Assignments

### `cmd/zcode-acp/main.go` (CLI entry — omp subcommand dispatch, `--model`, version stamp)

**Analog:** itself — the stdlib-flag block and `parseConfig`.

**Current flag/parse pattern** (lines 81-114):
```go
func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	appServerFlag := flag.String("app-server", "", ...)
	modelFlag := flag.String("model", "", "backend model selection as providerId/modelId — also "+envModel)
	...
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintln(os.Stdout, "zcode-acp "+version)
		return
	}
	// slog re-pointed to stderr BEFORE protocol starts (stdout = frames only)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	cfg, err := parseConfig(*appServerFlag, *modelFlag, *reasoningFlag, *apiKeyFlag)
	if err != nil {
		slog.Error("zcode-acp: invalid configuration", "err", err)
		os.Exit(exitInvalidConfig) // usage exit code 2, distinct from runtime exit 1
	}
```

**`--model` parse to reuse verbatim** (lines 152-164) — the omp `--model` binding is the SAME parse:
```go
	provider, model, ok := strings.Cut(selection, "/")
	if !ok || provider == "" || model == "" {
		return config{}, fmt.Errorf("%w, got %q", errModelShape, selection)
	}
	cfg.model = &acpfront.ModelSelection{ProviderID: provider, ModelID: model, ReasoningLevel: level}
```

**Sentinel-error pattern** (lines 71-79) — errors wrapped with dynamic context at the failure site, never constructed there (err113 lint):
```go
var (
	errLevelWithoutModel = errors.New("--reasoning-level set without --model")
	errModelShape        = errors.New("--model must be providerId/modelId")
	...
)
```

**Version stamp site** (line 36): `var version = "0.0.0-dev"` — Phase 7 must make `--version` print semver ≥ 18.1.9 (Pitfall 2: `0.0.0` parses as `tooOld` → Orca refuses the card). Exact stamp mechanics (bare `18.1.9` vs build-metadata) are planner discretion; the probe answers at root AND under `omp`, any position.

**Key semantics for the omp dispatch (Pitfall 3):** stdlib `flag.Parse` stops at the first non-flag argument — on `zcode-acp omp acp --model X` the `--model` currently lands in `flag.Args()` silently ignored. The omp arm must re-tokenize post-`omp` args (pre-tokenizing `flag.FlagSet` pass or cobra — planner decides; cobra was recorded in the STACK table 2026-11-19 but never adopted, and the `tidy-diff` mise gate at `mise.toml:52-57` asserts exactly ONE require line, so adopting cobra means updating that gate expectation).

**Test analog** — `cmd/zcode-acp/main_test.go` `TestWiringCodingPlanSeams` (:102, constructs production machinery lazily, asserts wiring) and `TestAgentBinaryEndToEnd` (:120, builds the real binary once per run into t.TempDir, drives initialize → session/new → prompt → cancel over real stdio with `--app-server` override onto a fake scenario). The omp parse unit tests follow the first; an end-to-end `omp acp` launch row can extend the second's build-once harness.

---

### `internal/acpport/agent.go` (adapter — delete arm, fork arm, detail-carrying refusals, IN-02)

**Analog:** itself — the delegation arms and the error seam.

**The delegation shape the new `DeleteSession` arm copies** (`CloseSession`, lines 402-415):
```go
func (a *agentAdapter) CloseSession(ctx context.Context, params acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	if err := a.h.CloseSession(ctx, string(params.SessionId)); err != nil {
		return acp.CloseSessionResponse{}, wireError(err)
	}
	return acp.CloseSessionResponse{}, nil
}
```
Phase 7 replaces (lines 417-423) the current `return acp.DeleteSessionResponse{}, unadvertised(...)` with: delegate to the front (close semantics where registered; immediate `{}` otherwise) and answer the empty `acp.DeleteSessionResponse{}`.

**The -32601 default pattern that stays for list/set_mode/set_model** (lines 554-560):
```go
func unadvertised(method string) error {
	return acp.NewMethodNotFound(method)
}
```
The optional explicit fork arm (if planner chooses it over the SDK default) copies this shape: `UnstableForkSession` returning `unadvertised(acp.AgentMethodSessionFork)` — byte-identical to the SDK dispatcher's answer (`agent_gen.go:191-206` in the module cache: `impl, ok := a.agent.(AgentUnstableForkSession); if !ok { return nil, NewMethodNotFound(method) }`).

**wireError — the coded-envelope seam** (lines 541-552). NOTE the returned error stays UNWRAPPED (SDK type-asserts `*RequestError`, no `errors.As`):
```go
func wireError(err error) error {
	if err == nil { return nil }
	var seamErr *Error
	if errors.As(err, &seamErr) {
		return acp.NewRequestError(seamErr.Code, seamErr.Message, nil) // ← data always nil today
	}
	return acp.NewInternalError(err.Error())
}
```
Phase 7 widens this: the seam `Error` (types.go:333-341) gains a details field, and a new constructor carries `data: {"details": ...}` so `/model` refusals ride `-32603` + `data.details` (Pitfall 1 — any other carriage is invisible to Orca's matchers).

**Capability advertisement stays unchanged** — `initializeResponse` (lines 170-198) builds `SessionCapabilities{Options: &acp.SessionOptionsCapabilities{}}`; delete/fork/list/load stay UNADVERTISED (Pitfall 4). The one honest-improvement candidate is advertising `sessionCapabilities.close` (implemented; Orca never checks it) — planner decision.

**IN-02 fix** — `ResumeSession` doc comment (lines 433-439) claims "the Handlers return value is validated non-empty" but line 444 discards the id (`if _, err := a.h.ResumeSession(...)`). Correct the comment (or add the check; the real guarantee lives in the front's `createSessionID`).

---

### `internal/acpport/types.go` (seam error vocabulary)

**Analog:** itself — the existing constructor family (lines 325-364):
```go
const (
	CodeInvalidRequest = -32600
	CodeInvalidParams  = -32602
	CodeInternal       = -32603
)

type Error struct {
	Code    int
	Message string
}

func NewInvalidRequest(format string, args ...any) *Error { ... }
func NewInvalidParams(format string, args ...any) *Error  { ... }
func NewInternal(format string, args ...any) *Error       { ... }

func (e *Error) Error() string { return e.Message }
```
The new detail-carrying constructor follows this exact shape (research's proposed name: `NewInternalWithData`), producing `{"code":-32603,"message":...,"data":{"details":"Unknown ACP model: <name>"}}` on the wire through the widened `wireError`. The Hb strings are byte-exact, two-source verified: prefix `Unknown ACP model: ` and `No model selected.` (both startsWith-matched by Orca's omp-dialect.ts).

---

### `internal/acpfront/front.go` (service — `/model` slash intercept, WR-01, WR-04, IN-04)

**Analog:** itself.

**`/model` intercept placement — the Prompt entry pattern** (lines 347-371): registry Get → isClosing gate → slot acquire. The intercept runs before slot acquisition (a refused `/model` never holds the turn slot):
```go
	st, err := f.registry.Get(params.SessionID)
	if err != nil {
		return "", acpport.NewInvalidRequest("session/prompt: %v: %s", err, params.SessionID)
	}
	if f.isClosing(params.SessionID) {
		return "", acpport.NewInvalidRequest("session/prompt: %s: the session is closing or closed", params.SessionID)
	}
	turnCtx, release, err := f.acquireTurn(params.SessionID, sink)
```

**Selector validation — the T-06-17 validate-first posture** (`SetSessionConfigOption`, lines 510-543): validate the untrusted value against the CURRENT confirmed option set BEFORE any backend call; unknown → coded error naming the allowed set:
```go
	raw, ok := f.latestSettings(sessionID)
	if !ok {
		return nil, acpport.NewInvalidParams(
			"session/set_config_option: %s: no confirmed settings to validate against (unknown session)", sessionID)
	}
	...
	def := configOptionByID(options, configID)
	if def == nil {
		return nil, acpport.NewInvalidParams(
			"session/set_config_option: unknown configId %q (allowed: %s)", configID, allowedOptionIDs(options))
	}
```
The `/model <name>` selector resolves the same way: the listed model values are the `providerId/modelId` serializations (`configoptions.go:296` — `Value: entry.Ref.ProviderID + "/" + entry.Ref.ModelID`). Match → apply via the setConfigMutation model arm; no match → the coded -32603 detail refusal. The reasoning level rides `confirmedLevel(s)` (lines 651-657) — the current confirmed level, never a fabricated one.

**The setModel wire payload the intercept and `--model` both reuse** (`setConfigMutation` model arm, lines 594-615 + `applyConfiguredModel`, lines 718-758):
```go
	model := map[string]any{wireKeyProviderID: providerID, wireKeyModelID: modelID}
	if level := confirmedLevel(s); level != "" {
		model["options"] = map[string]any{wireKeyReasoningLevel: level} // IN-04 omit-shape when empty
	}
	return map[string]any{
		wireKeySessionID: sessionID, optionIDModel: model, wireKeyPersistLastUsed: false,
	}, nil
```
`applyConfiguredModel` (718-758) already implements the whole launch-binding contract including the WR-04 orphan close on refusal — `--model` at launch reuses it unchanged.

**WR-01 fix shape** (lines 549-557): the refused picker setModel path currently calls the wire-only `closeOrphan`, leaving the front session registered (zombie). The fix runs the full `Front.CloseSession` teardown (cancel-drain → backend close → registry Remove → CloseObservers → map drops, lifecycle.go:142-212) or marks closing + releases front state — one designed teardown contract.

**WR-04 fix shapes** (three anchors): `front.go:272` — `f.recordCwd(sid, params.Cwd)` happens before subscribe; the subscribe-refusal return (277-283) calls `closeOrphan` but never drops the cwd → add `dropCwd` there (closeOrphan is the natural single place). `front.go:289` — `f.recordSettings(sid, res)` before `applyConfiguredModel`; the refusal return leaks cwd+settings → same treatment. `lifecycle.go:101` — resume's `recordCwd` precedes the re-issued subscribe; the subscribe-refusal return (107-112) leaks it → drop on that return path.

**IN-04 fix** (lines 833-836): `recordSettings` carries a dead `if f.settings == nil` re-init (the only constructor initializes the map, front.go:200; never reassigned). Delete the nil check. NOTE the sibling lazy-init of `closing` in `CloseSession` (lifecycle.go:156-158) IS load-bearing — keep it.

**Orphan-close pattern for reference** (`closeOrphan`, lines 886-896): best-effort wire close + structural `slog.Warn` (session/stage/cause only), never a silent skip.

---

### `internal/acpfront/lifecycle.go` (service — WR-05 TOCTOU)

**Analog:** itself.

**The WR-05 window** (lines 122-133): resume's final `isClosing` re-check and `registry.Register` are separate lock acquisitions; close's marking (lifecycle.go:148-165) holds `closingMu` only for the map write:
```go
	// WR-03 (plan 05-10): the mandatory re-check immediately before Register ...
	if f.isClosing(sessionID) {
		return "", acpport.NewInvalidRequest("session/resume: %s: the session is closing or closed", sessionID)
	}
	machine := turn.NewMachine(f.conn, sid, turn.NewPollWatcher(f.conn, turn.PollConfig{}))
	f.registry.Register(core.Session{ID: sid, Machine: machine})
```
Fix shape (research-confirmed): hold `closingMu` across re-check + Register, or have close re-`Get` after marking and release whatever is registered. Lock-order discipline: one-way turnMu → bridge.mu exists (T-5-38); any new closingMu nesting must respect the established orderings.

**Close's marking section to stay atomic with** (lifecycle.go:148-165): check-already-closing → lazy init → mark `closing[sessionID]=true` → unlock, all under one `closingMu` section.

---

### `internal/hostsvc/snapshot.go` (IN-01 — single-read fix)

**Analog:** itself — `Build` (lines 139-146) reads once; `BuildAccount` (lines 185-197) reads the same file twice:
```go
	rev, err := readBuiltinRevision(install.BuiltinFile)   // read 1
	...
	rules, err := readBuiltinProviderRules(install.BuiltinFile) // read 2 — IN-01: stale pairing possible
```
Fix shape: one read + one decode; derive revision AND rules from the same bytes. Note `readBuiltinProviderRules` (line 261) already documents "the SAME zcode-builtin.json readBuiltinRevision opens" — the fix makes that literally true. Keep the coded no-rules error (lines 229-232) and the gate-may-only-omit rule (207-209) untouched.

---

### `internal/interact/keyringprobe.go` (IN-03 — classify order)

**Analog:** itself — the classify switch (lines 224-240):
```go
	switch {
	case ctx.Err() != nil:                                              // parent-cancel first
		return probeClassCancelled, ...
	case errors.Is(pctx.Err(), context.DeadlineExceeded):                // ← IN-03: checked BEFORE success
		return probeClassTimeout, ...
	case execErr == nil && len(bytes.TrimSpace(stdout)) > 0:             // success arm
		return probeClassOK, nil
	...
	}
```
A boundary-timed success currently misclassifies as timeout. Fix: reorder (parent-cancel → success → deadline → marker → default) — or a comment documenting the fail-fast semantics; CONTEXT gives discretion (the fail-fast behavior is already correct). The four `ErrKeyring*` sentinels (lines 40-61) are the VM script's assertion targets — their exact text is the client-facing contract (AUTH-04); the VM script asserts on these strings, the script must not paraphrase them.

---

### `internal/acpfront/conformance_test.go` — the named conformance suite (QA-01, D-04)

**Analog:** `fullcycle_test.go` + `harness_test.go` + `golden_test.go` driver discipline.

**Composed harness entry** (`ComposeFront`, harness_test.go:410-460) — the in-process composition the named suite rides:
```
test client (SDK ClientSideConnection) ⇄ recording tee (TeeRW) ⇄ acpport connection ⇄ Front ⇄ zcode.Conn ⇄ fake.Serve(scenario)
```

**The arc shape to extend** (`TestFullCycle`, fullcycle_test.go:50-69) — one harness, one continuous client story, legs in the real client's order:
```go
	h := newFullCycleFront(t)
	ctx, cancel := h.ExchangeCtx()
	defer cancel()
	mustInitialize(ctx, t, h)
	fullCycleAuthenticate(ctx, t, h)
	sid := fullCycleOpenSession(ctx, t, h)
	fullCyclePrompt(ctx, t, h, sid)
	quietAfterResume := fullCycleResume(ctx, t, h, sid)
	fullCycleClose(ctx, t, h, sid, quietAfterResume)
	fullCycleTranscript(t, h, sid)
```
The named suite (planner names it; research suggests e.g. `TestConformanceV1Arc`) composes the same proven pieces PLUS the cancel leg (dedicated corpus rows `cancel-mid-turn`, `complete-then-cancel` already exist) into one byte-locked arc: initialize → session/new → prompt streaming → tool calls → permission decision → cancel.

**Front construction in tests** (fullcycle_test.go:78-103): `acpfront.New(conn, WithVersion(...), WithModel(...), WithAuth(...))` + `interact.NewResponder(interact.WithPermissionForwarder(...))` assigned to `conn.OnRequest` BEFORE any frame flows; client permission script set on `h.rec`.

**Byte-lock surface:** assertions ride `h.ACPLines()` (tee), `h.Updates()` (collector), and `h.Transcript(t)` (back-leg, closes the fake) — never internals. Tee-frame probes are substring constants (fullcycle_test.go:39-45).

**Pitfall 7 constraint:** delete/fork//model arms are ACP-front methods — their rows ride the COMPOSED harness (acpfront package), not `internal/zcode/fake` scenarios (that corpus is back-leg ZCode-dialect only; `"session/delete"` occurs ×0 on the back wire).

**Fail-loud corpus hygiene** (golden_test.go:365-398): every corpus scenario must have a replay driver — `t.Fatalf("scenario %s has no replay driver ...")`; goldens via `fake.CompareGolden` with normalizer hooks (golden_test.go:313-346) for machine-derived values.

---

### `scripts/vm-*.sh` (new — QA-02, D-05)

**Analog:** `experiments/check-docs.sh` (the repo's only established bash-script pattern) + `cmd/zcode-acp/main.go` `discoverInstall`.

**Bash-script conventions to copy** (check-docs.sh:1-14): `#!/usr/bin/env bash` shebang + a header comment naming purpose and output contract; explicit exit codes; pure-tool discipline (no dev toolchain on the VM — Pitfall 6: the script documents build-or-fetch for the static binary and asserts runtime deps only).

**What the script asserts** (research Pattern 7):
1. Desktop discovery (env/flag override → desktop entry → AppImage paths → deb `/opt/ZCode`) — the same order `main.go:463-490` drives; actionable nothing-found diagnostics naming every probe row.
2. Both auth paths configured (api-key env; coding-plan = desktop login, no key export).
3. `zcode-acp omp --version` probe + a scripted stdio ACP arc (initialize → session/new → one prompt) against the real binary.
4. The no-keyring error path — assert on the four coded sentinels from `internal/interact/keyringprobe.go:44-60` (verbatim strings: `ErrKeyringBinaryMissing`, `ErrKeyringNoDesktopSession`, `ErrKeyringNoEntry`, `ErrKeyringLookupTimeout`).
5. The paired provider-config env fact (`internal/supervisor/spawn.go` `ChildEnv` passes `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` AND `ZCODE_PERSONAL_PROVIDER_CONFIG_FILE` together) — assert the coding-plan path publishes account models by read-back, never the ack.

**Security shape (V2/V5, secret leakage):** the script fetches only checksum-verified official artifacts or uses an existing install; log lines never echo credential values (repo rule: env-var NAMES only in docs/scripts); scrub-first capture pipeline + `experiments/scan-secrets.sh` on tree AND staged diff before any evidence commit.

---

### `README.md` (mod — DOCS-01) and `docs/` additions (DOCS-02/03)

**Analog:** `README.md` itself + `docs/protocol/README.md`.

**README structure to preserve:** the current file's front matter (`# zcode-acp` + the honest one-paragraph "what it is") and the CI/branch-protection sections stay; Phase 7 inserts the user surface: what it is (honest ACP server for any ACP client), install, configuration (both auth paths), troubleshooting. English per PROJECT.md.

**docs/ conventions enforced by the repo's own gate** (`experiments/check-docs.sh` checks a-f): the heading `## Provenance legend` appears exactly once across docs/ and only in `docs/protocol/README.md` (other docs reference it); every `[V-EXP <id>]` tag must resolve to `experiments/captures/<id>*`; tag grammar `[V-SRC|V-EXP|V-RPT] <arg>`; no user-home absolute paths anywhere in committed docs/prose. New docs (launch recipe, protocol notes, limitations) MUST follow this grammar or the gate fails CI. Provenance content to state (D-07): no patching, no attestation forgery, PAT signed by the official binary, Apache-2.0 attribution for ported reference-adapter code (per `docs/research/reference-adapters.md`), known limitations (no quota counter over OAuth; Start Plan not headless).

**Launch recipe inputs (DOCS-02, all pinned in research Pattern 8):** per-agent command override `omp` = `<path>/zcode-acp omp` (shell-quoted-tokenized; `~/` expanded; `VAR=val cmd` prefixes REJECTED — hence the PATH-shim or `agentDefaultEnv` mechanism for `ZCODE_ACP_API_KEY`); version floor 18.1.9+; `PI_NO_PTY=1` ignore; the settings-store persistence path stays a labeled known-unknown.

---

### `07-UAT.md` + `experiments/README.md` ledger row (QA-03 manual gate)

**Analog:** `.planning/phases/06-oauth-coding-plan-path-model-picker/06-UAT.md` — the exact recording shape: YAML front matter (status/phase/source/started/updated), one numbered test with expected/awaiting/result fields, human confirmation preceding the evidence commit, honest verdict recording. Ledger discipline from `experiments/README.md` ground rules: raw captures stay outside the repo, scrubber-only entry into `experiments/captures/`, `scan-secrets.sh` green on tree AND staged diff, V-EXP id resolution.

**Pre-verification analog:** `test/realdesktop/` guarded tests (`ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop ...`) — everything automatable before the user's manual run (the 06 precedent: the guarded harness ran first, the human gate confirmed).

## Shared Patterns

### Coded error envelope (never raw errors)
**Source:** `internal/acpport/types.go:325-364` (constructors) + `internal/acpport/agent.go:541-552` (`wireError`)
**Apply to:** every new ACP arm (delete, fork, /model refusals) in acpport and acpfront.
```go
return acp.DeleteSessionResponse{}, wireError(err) // seam errors code through; the *RequestError stays unwrapped
```
The `/model` refusal additionally needs the NEW detail-carrying constructor — `data.details` on code -32603 is the only carriage Orca's matchers read (Pitfall 1).

### Validate-first against confirmed state (T-06-17)
**Source:** `internal/acpfront/front.go:510-543` (`SetSessionConfigOption`)
**Apply to:** the `/model <name>` selector — resolve against `latestSettings` → `BuildConfigOptions` listed values only; never forward an unvalidated string to the backend; refusals use the exact Hb prefixes (`Unknown ACP model: `, `No model selected.`).

### Teardown discipline (one contract)
**Source:** `internal/acpfront/lifecycle.go:142-212` (`CloseSession` ordering), `front.go:886-896` (`closeOrphan`)
**Apply to:** WR-01 (refused setModel → full `CloseSession` teardown, not wire-only `closeOrphan`), WR-04 (drop cwd/settings on all three post-create failure paths), the new delete arm (registered id → `CloseSession` semantics; unknown id → immediate `{}`).

### Byte-lock via corpus/tee
**Source:** `internal/zcode/fake/golden_test.go:365-398` (TestGolden fail-loud drivers, CompareGolden), `internal/acpfront/harness_test.go` (TeeRW), `fullcycle_test.go` (tee/transcript asserts)
**Apply to:** delete `{}` frame, fork `-32601 {"method":"session/fork"}` frame, `/model` refusal frames, the named conformance arc — byte-exact, never paraphrased.

### Honest errors / no fake success
**Source:** repo-wide (`cmd/zcode-acp/main.go:71-79` sentinel discipline; `keyringprobe.go` coded sentinels; auth read-back chain)
**Apply to:** version stamp (probe-passing semver ≥ 18.1.9 — load-bearing for QA-03), CLI argv errors (honest parse errors, unknown flags tolerated-but-ignored per the omp contract), VM script assertions (read-back, never the ack).

### Bash gate script shape
**Source:** `experiments/check-docs.sh:1-14`
**Apply to:** `scripts/vm-*.sh` — shebang, purpose header, declared output contract, explicit exit codes, no toolchain assumption on the VM.

### Doc conformance gates
**Source:** `experiments/check-docs.sh` (checks a-f), `experiments/scan-secrets.sh`, `docs/protocol/README.md` (provenance legend)
**Apply to:** README rewrite and all new docs/ files — provenance tag grammar, single legend definition site, no user-home absolute paths, secret-shape gate on tree + staged diff.

### Manual-tier recording
**Source:** `.planning/phases/06-oauth-coding-plan-path-model-picker/06-UAT.md`, `experiments/README.md` ground rules
**Apply to:** QA-03 (07-UAT.md + ledger row; user gate never autonomous) and QA-02 (one documented VM execution as field proof).

## No Analog Found

None — every file maps to an in-repo analog. Two role-match notes (not gaps):
- `scripts/vm-*.sh` is the first script outside `experiments/`; it copies the experiments bash-gate shape and documents the discovery order from `main.go`/`install-discovery.md`.
- The omp CLI subcommand dispatch has no existing subcommand analog in the binary — it extends `main.go`'s own flag/parse patterns; the research pins the two compliant options (pre-tokenizing `flag.FlagSet` ~50 lines vs cobra + `tidy-diff` gate update).

## Metadata

**Analog search scope:** `cmd/`, `internal/{acpport,acpfront,hostsvc,interact,supervisor,core,zcode/fake}`, `experiments/`, `docs/`, `test/realdesktop/`, `.planning/phases/06-*/`
**Files scanned:** ~25 (14 read in full or targeted, plus listings of fake testdata corpus (34 scenarios), docs/ tree, tools/ (empty), experiments/)
**All analogs verified git-tracked:** yes (git ls-files, 2026-11-22)
**Pattern extraction date:** 2026-11-22
