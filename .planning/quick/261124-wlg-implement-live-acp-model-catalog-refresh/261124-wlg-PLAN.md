---
phase: 261124-wlg
plan: 01
type: tdd
wave: 1
depends_on: []
files_modified:
  - internal/acpport/types.go
  - internal/acpport/sink.go
  - internal/acpport/agent.go
  - internal/acpport/catalogupdate_test.go
  - internal/acpfront/catalogrefresh.go
  - internal/acpfront/front.go
  - internal/acpfront/stream.go
  - internal/acpfront/catalogrefresh_internal_test.go
  - internal/acpfront/catalogrefresh_test.go
  - docs/protocol/acp-client-dialect.md
  - docs/protocol/app-server-dialect.md
autonomous: true

must_haves:
  truths:
    - "After the backend provider catalog changes, every active ACP session receives a session/update config_option_update carrying the complete confirmed configOptions list."
    - "A refresh acknowledgement alone never publishes a catalog; a fresh session/create settings.model.available reply is the read-back authority."
    - "A session keeps its confirmed current model when that model remains listed; if it disappears, an update is published only with the fresh backend-confirmed current/lastUsed fallback when that fallback is listed."
    - "Malformed, unconfirmed, or unlisted current values produce no fabricated selection and no unsafe update."
    - "A closing or closed session cannot pass the final publication gate, receive a catalog update, or regain a confirmed-settings entry."
    - "A failed client write does not replace the prior confirmed snapshot, so an identical later refresh remains eligible to retry."
    - "Registry notifications are classified off the backend read loop, coalesced single-flight, and every refresh worker has a bounded context-owned lifetime."
    - "The neutral acpport seam carries config-option DTOs only; internal/acpfront never imports the ACP SDK."
    - "Exact wire tests prove config_option_update framing and an integration test proves one backend catalog change reaches two active sessions."
  artifacts:
    - path: "internal/acpport/types.go"
      provides: "Neutral NoteConfigOptions kind and ConfigOptions payload"
      contains: ["NoteConfigOptions", "ConfigOptions"]
    - path: "internal/acpport/sink.go"
      provides: "SDK translation from neutral config-option notes to SessionConfigOptionUpdate"
      contains: ["config_option_update", "configOptionSelects"]
    - path: "internal/acpport/agent.go"
      provides: "Construction-time asynchronous sink binding and post-authenticate refresh request"
      contains: ["UpdateSinkBinder", "CatalogRefresher", "BindUpdateSink", "RequestCatalogRefresh"]
    - path: "internal/acpfront/catalogrefresh.go"
      provides: "Registry-event classification, bounded coalesced refresh, create read-back, current-model safety, and active-session fan-out"
      contains: ["RefreshRegistry", "SessionCreate", "mergeCatalogEntries", "NoteConfigOptions"]
    - path: "internal/acpfront/catalogrefresh_internal_test.go"
      provides: "Registry-patch classification, confirmed-current/fallback safety, close-during-fanout, and failure-then-identical-refresh tests"
    - path: "internal/acpfront/catalogrefresh_test.go"
      provides: "Composed backend-to-two-ACP-session integration and exact wire-frame assertions"
    - path: "docs/protocol/acp-client-dialect.md"
      provides: "Client-facing live config-option update contract"
    - path: "docs/protocol/app-server-dialect.md"
      provides: "Backend refresh/read-back and no-ack-trust contract"
  key_links:
    - from: "internal/acpport/agent.go"
      to: "internal/acpfront/front.go"
      via: "NewAgentSideConnection calls UpdateSinkBinder.BindUpdateSink before traffic; Front stores the connection-bound sink atomically"
    - from: "internal/acpfront/stream.go"
      to: "internal/acpfront/catalogrefresh.go"
      via: "observeEnvelope classifies state.updated patch.providers registry changes and calls RequestCatalogRefresh without blocking the read loop"
    - from: "internal/acpfront/catalogrefresh.go"
      to: "internal/hostsvc/snapshot.go"
      via: "RefreshRegistry runs provider/refreshRegistry; its reply is transport evidence only"
    - from: "internal/acpfront/catalogrefresh.go"
      to: "internal/acpfront/configoptions.go"
      via: "ParseSettings and BuildConfigOptions rebuild the full list; mergeCatalogEntries preserves a still-listed session selection over the fresh catalog"
    - from: "internal/acpfront/catalogrefresh.go"
      to: "internal/acpfront/front.go"
      via: "publishCatalogUpdate takes closingMu, rechecks closing plus registry membership, performs the bounded sink write, and only then records settings under settingsMu; CloseSession uses the same closingMu mark/drop ordering"
    - from: "internal/acpfront/catalogrefresh.go"
      to: "internal/acpport/sink.go"
      via: "NoteConfigOptions carries the full neutral list through the bound UpdateSink to session/update"
    - from: "internal/acpfront/catalogrefresh_test.go"
      to: "internal/acpfront/catalogrefresh.go"
      via: "ComposeFront drives a registry notification through the real Front, acpport connection, SDK client, and fake backend"
---

<objective>
Implement live ACP model-catalog refresh in zcode-acp: when the backend provider
model catalog changes, publish the complete confirmed `configOptions` array to
every active ACP session as `session/update` with
`sessionUpdate:"config_option_update"`.

Purpose: Orca 1.4.271 already adopts this notification, so the wrapper must stop
limiting config options to `session/new` and `session/set_config_option`
responses. The backend refresh acknowledgement is not proof of application; the
fresh `session/create` reply's `settings.model.available` is the confirmed
catalog surface.

Output: a neutral config-option update note, construction-time asynchronous sink
binding, a bounded event-driven registry refresh/read-back/fan-out path, current-
model safety, exact wire tests, a two-active-session integration test, and
protocol documentation. Do not modify Orca, dependencies, or go.mod.
</objective>

<execution_context>
@$HOME/.zcode/gsd-core/workflows/execute-plan.md
@$HOME/.zcode/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.claude/CLAUDE.md
@internal/acpport/types.go
@internal/acpport/sink.go
@internal/acpport/agent.go
@internal/acpport/agent_test.go
@internal/acpfront/front.go
@internal/acpfront/stream.go
@internal/acpfront/configoptions.go
@internal/acpfront/auth.go
@internal/acpfront/harness_test.go
@internal/acpfront/catalog_emit_test.go
@internal/acpfront/auth_test.go
@internal/hostsvc/readback.go
@internal/hostsvc/snapshot.go
@internal/zcode/fake/scenario.go
@docs/protocol/acp-client-dialect.md
@docs/protocol/app-server-dialect.md

Verified planning facts:

- `github.com/coder/acp-go-sdk v0.20.0` provides
  `acp.SessionConfigOptionUpdate` and `acp.SessionUpdate.ConfigOptionUpdate`.
- Current config options are emitted only in `session/new` and
  `session/set_config_option` responses; `SessionNote` has no config-option kind.
- The adapter holds the connection-bound `UpdateSink` but passes it only into
  `Prompt`. `PermissionBinder` is the existing construction-time binding
  precedent for front-originated asynchronous notifications.
- `provider/refreshRegistry` may acknowledge without emitting `state.updated`.
  A fresh raw `session/create` reply exposes the confirmed model list through
  `settings.model.available`; the existing auth probe shows the required
  create/close shape and host-obligation handling.
- `Front.observeEnvelope` currently ignores every non-`session/event`
  notification. Registry changes arrive as `state.updated` with
  `patch.providers`; prompt-status patches on the same method are not catalog
  changes.
- `settings` is already a bounded per-session confirmed-snapshot store dropped
  during teardown. Snapshot its keys and confirm each id with `registry.Get`
  rather than adding another session registry.

Current-model policy for this task: preserve a session's confirmed current (or
lastUsed) when it is still present in the fresh catalog. If it disappeared, use
the fresh probe's confirmed current/lastUsed only when that fallback is itself
listed. Otherwise retain the old snapshot and publish nothing for that session.
Never copy a requested value, infer a fallback, or manufacture `currentValue`.

TDD evidence convention: capture the complete real `go test` output and semantic
assessment in `.planning/tdd/`. If `gsd_run` is on PATH, require
`gsd_run check tdd-red-evidence <record> --raw` to return `RED_EVIDENCE_OK`.
If it is unavailable, preserve the raw report and record that limitation, as the
repository's recent quick-task precedent does; never fabricate a classifier
verdict. A compile failure is not RED evidence.
</context>

## Source coverage audit

| Source | Item | Plan coverage | Status |
|--------|------|---------------|--------|
| GOAL | Publish full configOptions through `config_option_update` when the backend catalog changes | Tasks 1-2 | COVERED |
| REQ | No ROADMAP requirement ids are assigned to this quick task | N/A | COVERED |
| RESEARCH | SDK update type exists; Orca 1.4.271 adopts it; ack is not read-back; `state.updated` is not guaranteed | Tasks 1-3 | COVERED |
| CONTEXT | Preserve the neutral DTO seam and reuse read-back/catalog-merge patterns | Tasks 1-2 | COVERED |
| CONTEXT | Confirmed registry read-back, active-session fan-out, and current-model safety | Task 2 | COVERED |
| CONTEXT | Bounded, single-flight, context-owned background work | Task 2 | COVERED |
| CONTEXT | Exact wire tests plus backend-to-all-active-sessions integration proof | Tasks 1-2 | COVERED |
| CONTEXT | No Orca, go.mod, dependency, or primary-checkout changes | Tasks 1-3 | COVERED |

<tasks>

<task type="tracer" tdd="true">
  <name>Task 1: Publish one neutral config-option note through the connection-bound sink</name>
  <files>internal/acpport/types.go, internal/acpport/sink.go, internal/acpport/agent.go, internal/acpport/catalogupdate_test.go</files>
  <behavior>
    - A neutral `NoteConfigOptions` carrying model and thought-level definitions becomes exactly one `session/update` notification with `sessionUpdate:"config_option_update"` and the complete `configOptions` array.
    - The SDK injects each `type:"select"` discriminant and preserves option order, category, currentValue, names, and descriptions.
    - `NewAgentSideConnection` binds the same connection sink into a front that opts into `UpdateSinkBinder`, before any frame flows.
    - A successful `Authenticate` asks an optional `CatalogRefresher` to refresh; a failed authenticate does not.
  </behavior>
  <action>
    RED, compile-first: add only the inert neutral vocabulary to
    `internal/acpport/types.go`: `NoteConfigOptions SessionNoteKind =
    "config_options"` and `ConfigOptions []ConfigOptionDef` on `SessionNote`.
    Update the SessionNote comment so this backend-confirmed snapshot note is
    traceable rather than synthesized. Do not change sink or adapter behavior
    yet; this keeps the planned RED tests compiling.

    Add `internal/acpport/catalogupdate_test.go` in package `acpport_test`,
    reusing the existing `scriptedHandlers`, pipe, frame-drain, and timeout
    helpers from `agent_test.go`. Define a small embedding handlers double with
    `BindUpdateSink(acpport.UpdateSink)` and
    `RequestCatalogRefresh(context.Context)` methods that record their inputs.
    Add these tests:

    1. `TestConfigOptionUpdateWire` constructs a real agent-side connection and
       calls its bound sink with one `NoteConfigOptions` carrying the existing
       model/thought-level fixture vocabulary. Assert the raw frame is byte-exact:
       `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"sess_1","update":{"configOptions":[{"category":"model","currentValue":"zai-api/GLM-5.3-Flash","id":"model","name":"Model","options":[{"description":"The flagship reasoning model","name":"GLM-5.3-Flash","value":"zai-api/GLM-5.3-Flash"},{"name":"GLM-5.3","value":"zai-api/GLM-5.3"}],"type":"select"},{"category":"thought_level","currentValue":"high","id":"thought_level","name":"Thought level","options":[{"description":"deep reasoning","name":"High","value":"high"},{"name":"Low","value":"low"}],"type":"select"}],"sessionUpdate":"config_option_update"}}}`.
       At RED this must fail because the sink rejects the new note kind, not
       because of a build or fixture error.
    2. `TestAsyncSinkBinding` constructs through `NewAgentSideConnection` and
       asserts the handlers double received the non-nil connection sink before
       the constructor returned.
    3. `TestAuthenticateRequestsCatalogRefresh` proves one refresh request after
       a successful authenticate and zero additional requests after an
       authenticate error.

    Run `go test -race -count=1 -run 'TestConfigOptionUpdateWire|TestAsyncSinkBinding|TestAuthenticateRequestsCatalogRefresh' ./internal/acpport`.
    Save the complete output and semantic assessment as
    `.planning/tdd/261124-wlg-task1-red.json`; the primary target must fail on
    the unknown-note assertion. Apply the evidence protocol in the context
    above, then commit the inert DTO vocabulary and failing tests as
    `test(261124-wlg): add failing live config-option update contracts`.

    GREEN: in `internal/acpport/sink.go`, translate `NoteConfigOptions` through
    the existing `configOptionSelects` helper into
    `acp.SessionConfigOptionUpdate{SessionUpdate:"config_option_update",
    ConfigOptions: selects}`. Use a non-nil empty slice if no definitions are
    supplied so the required array is `[]`, never `null`. Keep all SDK types in
    this package.

    In `internal/acpport/agent.go`, add exported opt-in interfaces analogous to
    `PermissionBinder`: `UpdateSinkBinder` with `BindUpdateSink(UpdateSink)` and
    `CatalogRefresher` with `RequestCatalogRefresh(context.Context)`. After
    `bindSink`, forward the same sink to `UpdateSinkBinder`. After a successful
    `Handlers.Authenticate`, invoke `CatalogRefresher`; an authenticate error
    returns before the refresh request. Do not change the authenticate response
    or make a catalog-notification failure turn confirmed auth into a failure.

    Run the focused tests and then `go test -race -count=1 ./internal/acpport`.
    Refactor only if the focused and package tests stay green. Commit as
    `feat(261124-wlg): publish config option updates through the bound sink`.
  </action>
  <verify>
    <automated>go test -race -count=1 -run 'TestConfigOptionUpdateWire|TestAsyncSinkBinding|TestAuthenticateRequestsCatalogRefresh' ./internal/acpport && go test -race -count=1 ./internal/acpport && golangci-lint run ./internal/acpport/...</automated>
  </verify>
  <done>The exact config_option_update frame is locked; the connection-bound sink reaches the front asynchronously; successful authenticate requests a catalog refresh while failed authenticate does not; all acpport tests, the SDK import-boundary test, and lint pass; RED and GREEN evidence is preserved without a compile-failure RED.</done>
</task>

<task type="tdd" tdd="true">
  <name>Task 2: Refresh from a confirmed create read-back and fan out safely to active sessions</name>
  <files>internal/acpfront/catalogrefresh.go, internal/acpfront/front.go, internal/acpfront/stream.go, internal/acpfront/catalogrefresh_internal_test.go, internal/acpfront/catalogrefresh_test.go</files>
  <behavior>
    - Only `state.updated` envelopes carrying a registry `patch.providers` object request a catalog refresh; prompt-status and malformed patches do not.
    - A registry change runs `provider/refreshRegistry`, then a raw probe `session/create`, then best-effort `session/close`; no notification is derived from the refresh acknowledgement.
    - Two active ACP sessions with different prior selections both receive the fresh full list; a still-listed selection is preserved and a disappeared selection uses only a listed backend-confirmed fallback.
    - No current/fallback, an unlisted fallback, malformed settings, or an unlisted rendered currentValue suppresses that session's update and retains its prior snapshot.
    - Duplicate identical registry signals during one refresh coalesce to one backend refresh; unrelated notifications and slotless/closed sessions cause no catalog traffic or update.
    - Final publication is serialized with CloseSession's closing mark and registry/settings teardown, without holding settingsMu across client I/O.
    - A failed client write leaves the previous confirmed snapshot intact, so the same catalog remains eligible on a later refresh.
  </behavior>
  <action>
    RED, compile-first: create `internal/acpfront/catalogrefresh.go` with only
    three inert signatures so tests compile and fail on assertions:
    `registryCatalogPatch(raw json.RawMessage) bool` returning false,
    `snapshotForLiveCatalog(fresh, previous json.RawMessage) (json.RawMessage, bool)`
    returning `(nil, false)`, and
    `publishCatalogUpdate(ctx context.Context, sessionID string, merged
    json.RawMessage, options []acpport.ConfigOptionDef) bool` returning false.
    Add no routing or worker behavior yet.

    Add `internal/acpfront/catalogrefresh_internal_test.go` in package
    `acpfront`. `TestRegistryCatalogPatchClassification` covers a registry-scoped
    `patch.providers` payload, a `prompt_started` status patch, a registry patch
    with no providers, and malformed JSON. `TestSnapshotForLiveCatalogSafety`
    uses small raw settings fixtures to cover: previous current remains listed
    (preserve it over the fresh probe's different current); previous current
    disappears and fresh current is listed (use the confirmed fallback);
    previous current remains listed while the probe has no current (still safe);
    disappeared current with no probe current; disappeared current with an
    unlisted probe current; and malformed fresh settings. Assert both the
    returned snapshot's parsed current and its `BuildConfigOptions` result.

    Add deterministic publication-state tests in the same in-package file.
    `TestCatalogPublicationCloseGate` has two arms. The closed-session arm
    registers then removes a registry entry, drops its settings, and calls
    `publishCatalogUpdate` with the stale candidate; it must return false, call
    the sink zero times, and leave settings absent. The close-during-fanout arm
    uses `newInterleaveFront`, `openInterleaveSession`, and
    `interleaveScenario(0)` from `lifecycle_internal_test.go`, records an old
    snapshot, and binds a sink that signals entry then blocks on a release
    channel. Start publication, wait for sink entry, start `CloseSession`, and
    prove `closingMu.TryLock()` cannot succeed while the bounded publication
    write is in flight. Release the sink; publication records the new snapshot
    and unlocks before close can mark/remove/drop. Assert exactly one update,
    successful close, and no settings entry after teardown.
    `TestCatalogPublicationFailureThenIdenticalRefresh` uses a registered test
    session and a sink that fails once then succeeds. The first
    `publishCatalogUpdate` returns false and leaves the old snapshot byte-intact;
    a second call with the identical merged snapshot/options returns true, calls
    the sink twice, and records the new snapshot. This locks retry eligibility
    without a separate published-state cache.

    Add `internal/acpfront/catalogrefresh_test.go` in package `acpfront_test`.
    Build one in-code fake scenario with ordered `Times:1` create rules for
    active session A (old model A), active session B (old model B), and the
    refresh probe. Subscribe and close accept their normal shapes. A prompt on
    session A emits the same registry `state.updated` patch twice, then a
    turn-completed session event so the prompt returns. The registry patch lists
    model B and new model C but carries no current selection. The refresh rule
    delays its reply, returns only `{"refreshed":true}`, and emits no
    `state.updated`. The probe create returns a distinct session id and a full
    settings snapshot whose current is B and whose available entries are B and
    C with fresh labels. `TestLiveCatalogRefresh` opens A and B, prompts A,
    bounded-polls the raw ACP tee for both sessions' config-option frames, and
    compares each frame byte-for-byte against the full expected JSON-RPC
    notification. The expected model option for both sessions is currentValue
    `zai-api/GLM-5.3`, with fresh-labelled B then C entries and no thought-level
    option. Assert the backend transcript has exactly one refresh, three creates
    (two active plus one probe), and refresh before probe-create before
    probe-close. This proves fan-out, fresh labels, fallback safety, duplicate
    coalescing, and that the ack/patch alone was not the catalog source.

    Run `go test -race -count=1 -run 'TestRegistryCatalogPatchClassification|TestSnapshotForLiveCatalogSafety|TestCatalogPublicationCloseGate|TestCatalogPublicationFailureThenIdenticalRefresh|TestLiveCatalogRefresh' ./internal/acpfront`.
    Save complete output and semantic assessment as
    `.planning/tdd/261124-wlg-task2-red.json`; tests must compile and fail on the
    planned false/no-notification assertions. Apply the evidence protocol and
    commit as `test(261124-wlg): add failing confirmed catalog refresh tests`.

    GREEN: implement `internal/acpfront/catalogrefresh.go` around the existing
    patterns, not a parallel DTO or SDK path:

    1. Add `BindUpdateSink` to Front and store the sink in an
       `atomic.Pointer[acpport.UpdateSink]`. Add mutex-guarded
       `catalogRefreshRunning`, `catalogRefreshPending`, and
       `catalogRefreshSignal` fields plus the bound sink field to `Front` in
       `front.go`. Do not store a `context.Context` in the struct.
    2. Implement `RequestCatalogRefresh(ctx)` as the adapter-facing wrapper over
       `requestCatalogRefresh(ctx, signal string)`. Return on cancellation or
       when no active confirmed session exists. Registry events pass a SHA-256
       digest of their raw params as the signal; explicit post-authenticate
       refresh passes an empty signal. Under the mutex, deduplicate an identical
       non-empty signal, mark pending if a worker is already running, otherwise
       start one worker. The worker owns a single bounded
       `context.WithTimeout(context.Background(), catalogRefreshBudget)` (use
       15 seconds), runs at most while that context is alive, coalesces pending
       signals, and always clears running/pending before exit. A failed pass
       clears the remembered signal so a later event can retry. There is no
       ticker, polling fallback, or per-notification unbounded goroutine.
    3. In `refreshCatalogOnce`, call `hostsvc.RefreshRegistry`, then perform the
       raw read-back probe: `os.MkdirTemp`, bare `session/create` with the same
       workspace path/key shape as `probeCreateReadback`, parse the id with
       `createSessionID`, and best-effort `session/close`; remove the temp dir.
       A create/parse failure returns an error and publishes nothing. A close
       failure logs structurally but does not discard a valid read-back.
    4. Snapshot active ids from the `settings` map under `settingsMu`, then keep
       only ids resolved by `registry.Get` and not `isClosing`; sort the ids for
       deterministic fan-out. Do not add a second registry or hold the settings
       lock during backend/client calls.
    5. For each active id, read its previous snapshot and call
       `snapshotForLiveCatalog`. Reuse `ParseSettings`, `BuildConfigOptions`,
       and `mergeCatalogEntries`: fresh availability is authoritative; a
       previous current/lastUsed still listed is preserved over the fresh
       probe's current; otherwise the fresh current/lastUsed must be non-nil and
       listed. Reject the update if every rendered option's currentValue is not
       a member of its own listed values. Compare old and new
       `BuildConfigOptions` with `reflect.DeepEqual` and skip unchanged lists.
    6. Implement `publishCatalogUpdate` as the closing-aware final gate. Load
       the bound sink first; a missing sink returns false. Take `closingMu` and
       keep it through the bounded sink write and the subsequent settings
       commit. Under that lock, reject a marked-closing session and re-resolve
       the candidate with `registry.Get`; this second check catches a session
       closed after candidate enumeration. Then emit one
       `acpport.SessionNote{SessionID:id, Kind:acpport.NoteConfigOptions,
       ConfigOptions:options}` through the sink. Only after the write succeeds,
       call `recordSettings` while still holding `closingMu`; this acquires
       `settingsMu` after client I/O, never across it. A missing sink or write
       error logs structurally (session id and error only), returns false
       without changing settings, so the old/new comparison on an identical
       later refresh still sees an undelivered catalog and retries.
       The lock order is one-way `closingMu -> registry` and
       `closingMu -> settingsMu`; candidate collection releases `settingsMu`
       before checking closing state. Holding `closingMu` across the bounded
       write deliberately linearizes publication before CloseSession's mark:
       either publication commits first and close subsequently drops it, or
       close marks/removes first and publication emits nothing. Do not route
       session-level catalog updates through the active turn pump.
    7. In `stream.go`, before the existing non-`session/event` return, classify
       `zcode.MethodStateUpdated` with `registryCatalogPatch` and call
       `requestCatalogRefresh(context.Background(),
       registryCatalogSignal(env.Params))`. Classification, hashing, and the
       mutex hand-off are the only work done on the read goroutine.

    Run the focused tests, then `go test -race -count=1 ./internal/acpfront
    ./internal/acpport`. Fix only behavior needed by these contracts; preserve
    existing auth, stream, and lifecycle behavior. Refactor if needed and commit
    as `feat(261124-wlg): refresh and fan out confirmed model catalogs`.
  </action>
  <verify>
    <automated>go test -race -count=1 -run 'TestRegistryCatalogPatchClassification|TestSnapshotForLiveCatalogSafety|TestCatalogPublicationCloseGate|TestCatalogPublicationFailureThenIdenticalRefresh|TestLiveCatalogRefresh' ./internal/acpfront && go test -race -count=1 ./internal/acpfront ./internal/acpport && golangci-lint run ./internal/acpfront/... ./internal/acpport/...</automated>
  </verify>
  <done>A backend registry change causes one bounded refresh/read-back pass; both active ACP sessions receive exact full config_option_update frames from the fresh create snapshot; still-listed selections are preserved, disappeared selections use only a listed backend-confirmed fallback, and unsafe/malformed states publish nothing; the final closing-aware gate prevents closed-session updates/settings resurrection and serializes close-during-publication; failed writes retain retry eligibility; duplicate signals coalesce; existing front/acpport tests and lint are green; RED and GREEN evidence is preserved.</done>
</task>

<task type="auto">
  <name>Task 3: Document the live catalog contract and run the complete repository gate</name>
  <files>docs/protocol/acp-client-dialect.md, docs/protocol/app-server-dialect.md</files>
  <action>
    Update `docs/protocol/acp-client-dialect.md` with a concise section stating
    that the wrapper emits full `configOptions` through `session/update`
    `config_option_update` after a confirmed backend catalog refresh, that Orca
    1.4.271 adopts the notification, and that each active session gets its own
    update. Record the safety rule: preserve a still-listed session selection;
    otherwise publish only a backend-confirmed listed fallback, and otherwise
    keep the prior client state. Do not change Orca or invent provenance tags.

    Update `docs/protocol/app-server-dialect.md` section 8 to make the wrapper's
    live-refresh recipe explicit: classify registry `state.updated`
    `patch.providers` changes, call `provider/refreshRegistry`, treat its reply
    as receipt only, read `settings.model.available` from a fresh raw
    `session/create`, close the probe best-effort, then fan out. State that
    refresh alone does not guarantee `state.updated`, so the create read-back is
    mandatory, and that the worker is event-driven, single-flight, coalesced,
    and bounded by a 15-second context. Document the polling/watch fallback as
    deferred by this task's constraint: the shipped path is registry-event
    driven plus the explicit post-authenticate refresh request.

    Run `bash experiments/check-docs.sh`, then `mise run ci`, then
    `mise run tidy-diff`. The last command must leave go.mod/go.sum unchanged and
    confirms no dependency drift. Commit the two docs as
    `docs(261124-wlg): document confirmed live model catalog updates`.

    Perform the final scope review over history plus the worktree, not only the
    current diff. Use the observed feature-branch base
    `BASE=b5440a4ccfec0f26f14d120dc198aebe620c0621`, verify it is an ancestor of
    HEAD, and inspect `git log --oneline --reverse "$BASE"..HEAD`,
    `git diff --name-status "$BASE"..HEAD`, and `git diff --name-status`.
    Cross-check every task commit and both committed/uncommitted file lists
    against this plan's `files_modified` implementation allowlist. The plan,
    `.planning/tdd/261124-wlg-*.json` evidence records, and the quick-task
    `261124-wlg-SUMMARY.md` are required workflow artifacts outside the
    implementation-file scope when present; no other file is allowed. Run
    whitespace checks against both the branch-base range and the uncommitted
    diff.
  </action>
  <verify>
    <automated>bash experiments/check-docs.sh && mise run ci && mise run tidy-diff && BASE=b5440a4ccfec0f26f14d120dc198aebe620c0621 && git merge-base --is-ancestor "$BASE" HEAD && git log --oneline --reverse "$BASE"..HEAD && git diff --name-status "$BASE"..HEAD && git diff --name-status && git diff --check "$BASE"..HEAD && git diff --check</automated>
  </verify>
  <done>Both dialect documents state the same confirmed read-back/fan-out/safety contract; docs checks, build, vet, strict lint, full race suite, experiments, tidy-diff, and committed/uncommitted whitespace checks pass; every task commit and both file lists have been reviewed against the implementation allowlist with workflow artifacts classified separately; go.mod/go.sum and Orca are unchanged.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| backend app-server → Front | Untrusted `state.updated` patches and `session/create` settings cross the JSONL boundary; malformed or stale data must not become client state. |
| Front → ACP client | Session ids and complete config-option selections cross the SDK wire boundary; writes may fail or race session teardown. |
| backend event stream → worker lifecycle | Repeated or hostile registry notifications can create background work if not coalesced and bounded. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-261124-01 | Denial of service | `internal/acpfront/catalogrefresh.go` worker gate | medium | mitigate | Classify only registry provider patches; deduplicate identical signals; single-flight pending coalescing; one 15-second context-owned worker lifetime; no polling fallback; read-loop hand-off does no I/O. |
| T-261124-02 | Tampering / integrity | `internal/acpfront/catalogrefresh.go` snapshot merge and publication | medium | mitigate | Ignore the refresh ack as proof; require fresh create settings; preserve only listed prior selections; require any fallback to be backend-confirmed and listed; reject malformed/unlisted currentValue; compare before publishing. |
| T-261124-03 | Information disclosure | refresh/probe error logging | low | mitigate | Log only step, session id, and error; never log raw settings, credentials, prompt content, or full backend params; temp probe directories are removed. |
| T-261124-04 | Session-boundary violation | active-session fan-out versus close/delete | medium | mitigate | Final publication holds closingMu, rechecks closing and registry membership, writes while teardown cannot mark/remove, then records settings before unlocking; failed writes do not record; deterministic close-during-fanout and retry tests lock both arms. |
| T-261124-SC | Tampering / supply chain | package-manager installs | high | mitigate | No package installs or dependency changes are authorized; go.mod/go.sum must remain unchanged through `mise run tidy-diff`; any proposed install requires the package-legitimacy gate and a blocking human checkpoint. |
</threat_model>

<verification>
- Task 1's raw-frame test proves the exact SDK JSON-RPC notification, including the complete model and thought-level arrays.
- Task 1's binding and authenticate tests prove the asynchronous sink exists before traffic and refresh is requested only after successful auth.
- Task 2's internal tests prove registry-patch classification, all current/fallback safety arms, deterministic close-during-fanout behavior, and failure-then-identical-refresh retry eligibility.
- Task 2's composed test proves a backend catalog change reaches both active ACP sessions after refresh → create read-back → close, with one coalesced refresh and exact per-session wire frames.
- `go test -race -count=1 ./internal/acpport ./internal/acpfront` is green.
- `golangci-lint run ./internal/acpport/... ./internal/acpfront/...` is green.
- `bash experiments/check-docs.sh`, `mise run ci`, `mise run tidy-diff`, and committed/uncommitted `git diff --check` are green.
- Every task commit in `b5440a4ccfec0f26f14d120dc198aebe620c0621..HEAD` plus the uncommitted diff is reviewed against `files_modified`; planning/TDD/SUMMARY artifacts are classified separately as workflow outputs.
- `TestSDKImportBoundary` remains green; no SDK import is added outside `internal/acpport`.
- RED evidence shows assertion failures, not build/fixture failures; GREEN follows RED in git ancestry; any unavailable `gsd_run` classifier is disclosed rather than fabricated.
</verification>

<success_criteria>
- A live backend provider-model change produces one full `config_option_update` per active ACP session.
- The published catalog and every currentValue come from confirmed backend snapshots; an acknowledgement or requested value is never treated as confirmation.
- A disappeared selected model is replaced on the wire only by a listed fallback already confirmed by the fresh backend read-back; otherwise that session receives no unsafe update.
- A closing or closed session cannot receive a late fan-out update or regain settings, and a failed client write leaves the same catalog eligible for retry.
- Registry-driven background work is single-flight, coalesced, bounded, context-owned, and safely ends with the connection or deadline.
- The neutral DTO seam remains intact and the installed SDK version is unchanged.
- Exact wire tests, the two-session integration test, the full race suite, strict lint, docs checks, experiments, and tidy-diff all pass.
</success_criteria>

<output>
Create `.planning/quick/261124-wlg-implement-live-acp-model-catalog-refresh/261124-wlg-SUMMARY.md` when done, including RED/GREEN evidence paths, semantic assessments, commit hashes, and any unavailable-classifier disclosure.
</output>
