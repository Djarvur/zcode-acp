---
phase: 06-oauth-coding-plan-path-model-picker
reviewed: 2026-11-22T05:05:47Z
depth: standard
files_reviewed: 29
files_reviewed_list:
  - cmd/zcode-acp/main.go
  - cmd/zcode-acp/main_test.go
  - internal/acpfront/codingplan_cycle_test.go
  - internal/acpfront/configoptions.go
  - internal/acpfront/configoptions_test.go
  - internal/acpfront/front.go
  - internal/acpfront/initialize_test.go
  - internal/acpfront/lifecycle.go
  - internal/acpfront/lifecycle_test.go
  - internal/acpfront/newsession_test.go
  - internal/acpfront/setconfigoption_test.go
  - internal/acpfront/tool_stream_test.go
  - internal/acpport/agent.go
  - internal/acpport/agent_test.go
  - internal/acpport/types.go
  - internal/hostsvc/headers_pat.go
  - internal/hostsvc/headers_pat_test.go
  - internal/hostsvc/patminter_test.go
  - internal/hostsvc/snapshot.go
  - internal/hostsvc/snapshot_account_test.go
  - internal/interact/familysel.go
  - internal/interact/familysel_test.go
  - internal/interact/keyringprobe.go
  - internal/interact/keyringprobe_test.go
  - internal/interact/responder.go
  - internal/interact/responder_test.go
  - internal/zcode/fake/golden_test.go
  - test/realdesktop/apicycle_test.go
  - test/realdesktop/codingplan_test.go
findings:
  critical: 0
  warning: 6
  info: 4
  total: 10
  necessity: 1
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-11-22T05:05:47Z
**Depth:** standard
**Files Reviewed:** 29
**Status:** issues_found

## Summary

Reviewed all 29 Phase-6 files at standard depth (full per-file reads with language-specific Go checks), cross-referencing the imported modules the changed code depends on (`internal/hostsvc/patminter.go`, `internal/interact/credstore.go`, `internal/acpfront/auth.go`, `turnctx.go`, `stream.go`, `permission.go`, `internal/zcode/conn.go`) so every finding is verified against the real call graph rather than the file in isolation.

Overall assessment: this is a carefully built slice. The credential-hygiene posture (no token material in logs, digests-only fields, PAT struct with no renderers), the validate-before-wire discipline on `session/set_config_option`, the D-05 exactly-one re-mint, and the scope-gate-before-serve ordering are all implemented as documented and are test-locked. `go build ./...` and `go vet` on the reviewed packages are clean. No Critical findings: no injection surface (all subprocess argv comes from fixed, field-verified contracts; no user input reaches an argv), no hardcoded secrets (all fixture material is synthetic and scan-gated), no auth bypass (authenticate validates the method id against the advertised set; the pre-auth runtime-headers ask delegates to the honest unconfigured arm), and the concurrency hot spots (turn slot, dedupe table, pending-cancel takes) hold up under trace.

The six Warnings cluster around one theme: **error and transition paths that leave inconsistent state**. The refused-picker-mutation path closes the backend session but leaves the front-side session alive (WR-01); the reactive re-mint window mislabels concurrent asks as "login required" (WR-02, with a flake-prone test lock WR-03); post-create failure paths leak cwd/settings map entries the code's own comments claim stay bounded (WR-04); and the resume-vs-close race gate has a residual TOCTOU window that leaks a machine's observers (WR-05). One manual-tier test carries an unsynchronized recorder its own sibling file explicitly guards against (WR-06). The Info items are a builtin-table double-read (IN-01), a comment that claims a validation the adapter does not perform (IN-02), a boundary classification race in the keyring probe (IN-03), and one provable impossible-state guard (IN-04, necessity).

## Warnings

### WR-01: Refused picker setModel closes the backend session but leaves a front-side zombie session

**File:** `internal/acpfront/front.go:542-550`
**Issue:** When `session/setModel` refuses the config-option mutation, the handler calls `f.closeOrphan(ctx, sessionID, "set_config_option setModel", err)` — a wire-only `session/close` — and returns the coded error. But unlike the NewSession orphan path (whose comment correctly notes "the orphan deliberately holds no registry entry"), this session IS fully registered: the machine, its conn fan-out observers, the cwd entry, and the confirmed-settings entry all persist. The result is a zombie: the backend session is dead, while `registry.Get` still succeeds, `isClosing` is false, and the settings store still validates against a snapshot of a session that no longer exists. A subsequent prompt acquires the turn slot and sends `session/send` to a closed backend session (a confusing -32603), and every later picker ask re-runs the same close-orphan-and-fail cycle, logging a warn each time. The leaked registry entry, observers, and map entries are never released unless the client explicitly closes — and a client that just saw a -32603 has no reason to.
**Fix:** On the mutation-refusal path, either (a) run the full SESS-01 teardown instead of the wire-only close — call `f.CloseSession(ctx, sessionID)` (it already performs cancel-drain → backend close → registry Remove → observer removes → cwd/settings drops and tolerates a failed backend close) and return the mutation error; or (b) mark the session in the `closing` set and release the front-side state so post-refusal asks get the established "session is closing or closed" posture instead of re-failing against a dead backend.

### WR-02: Concurrent runtime-headers asks during the reactive re-mint window answer a misleading "login required"

**File:** `internal/hostsvc/headers_pat.go:99-106`
**Issue:** The mode-aware source reads the cache first: `pat, ok := deps.Minter.Cached(ctx)` and answers `ErrPatLoginRequired` when absent. But there is a real window in which the cache is legitimately empty while the user IS logged in: the D-05 reactive arm runs `Invalidate("rejected-fingerprint-match")` and then `Ensure(ctx)`, and in production `Ensure` performs a network mint chain (up to three HTTPS calls at a 15 s per-call bound). Any concurrent runtime-headers ask landing inside that window observes `ok == false` and answers the coded `project_token_login_required (log in via the ZCode desktop first)` — which the responder turns into a turn-killing error envelope telling an authenticated user to log in. The ask-side "never mints into an unauthenticated state" posture is correct; the problem is that the empty-cache answer cannot distinguish "no login exists" from "a re-mint for the existing login is in flight".
**Fix:** Track the in-flight mint on the minter surface (e.g. expose `Minting() bool`, or have `Cached` return a tri-state), and in the PAT-less arm either join the single-flight `Ensure` when a mint for the current login is already running, or answer a distinct transient coded class instead of `ErrPatLoginRequired`.

### WR-03: `testPatConcurrentArm` claims determinism its invalidate→ensure window cannot guarantee

**File:** `internal/hostsvc/headers_pat_test.go:560-620`
**Issue:** The test asserts every one of 7 plain concurrent asks answers a well-shaped success while exactly one fingerprint ask re-mints ("The deterministic concurrency shape"). It is not deterministic: between the fingerprint asker's `Invalidate` (sets `hasCache = false`) and its `Ensure` (re-installs the cache), a plain asker can call `Cached`, observe `ok == false`, and receive `ErrPatLoginRequired` — `werr != nil` → `badAsks++` → test failure. The script double's `Ensure` is fast, so the window is microseconds, but with 8 goroutines under `-race` scheduling can park the fingerprint asker between its two lock acquisitions indefinitely. This is the test-side face of WR-02, and it makes the suite flaky in exactly the run mode (`-race`) the file header advertises.
**Fix:** After fixing WR-02 (source-side), assert the new contract: concurrent asks during the window either join the mint or answer the transient class — never `ErrPatLoginRequired`. Until then, either widen the script double to hold the re-mint open and accept the transient class for window-landing riders, or serialize the window (fire the fingerprint ask, poll `invalidates == 1` before releasing the riders) so the assertion matches what the source actually guarantees.

### WR-04: Post-create failure paths leak cwd/settings entries the code's own bounding invariant claims are dropped

**File:** `internal/acpfront/front.go:272,289-292` and `internal/acpfront/lifecycle.go:101,107-112`
**Issue:** The WR-07 discipline is stated in both files: the cwd and confirmed-settings maps "stay bounded with the registry" — entries are dropped in `CloseSession` beside `dropCwd`/`dropSettings`. But three failure paths register entries for sessions that never obtain a registry entry, so nothing ever drops them:
- `NewSession` records the cwd (`front.go:272`) before the subscribe call; a subscribe refusal returns via `closeOrphan` without dropping it.
- `NewSession` records the settings snapshot (`front.go:289`) before `applyConfiguredModel`; a refused setModel follow-up returns without dropping cwd or settings.
- `ResumeSession` records the cwd (`lifecycle.go:101`) before the re-issued subscribe; a subscribe refusal returns without dropping it.
Each repeated session/new or resume failure leaks one map entry in a long-running process. The `closeOrphan` helper is the natural single place to close this gap for the first two paths.
**Fix:** In `closeOrphan`, after the best-effort backend close, call `f.dropCwd(sessionID)` and `f.dropSettings(sessionID)`; on `ResumeSession`'s subscribe-refusal return, drop the recorded cwd. (Dropping is safe: no registry entry or turn can reference these ids afterward — the machine was never registered.)

### WR-05: Resume-vs-close race gate has a residual TOCTOU window that leaks a machine's observers

**File:** `internal/acpfront/lifecycle.go:118-133` (interleaved with `CloseSession`, `lifecycle.go:142-211`)
**Issue:** WR-03's fix (plan 05-10) re-checks `isClosing(sessionID)` immediately before `registry.Register` — but the check and the Register are not atomic with the closer's marking, so the leak arm the re-check was built to close is narrowed, not eliminated. Interleaving: resume passes its re-check at t0 → close runs `registry.Get` (machine A) at t1 → close marks the closing set at t2 → resume Registers machine B at t3 > t2 → close's teardown then `registry.Remove(sessionID)` (removing B's entry) and `st.Machine.CloseObservers()` (closing only A's observers). Machine B's notification and router fan-out entries stay on the conn for the process lifetime — exactly the "zombie machine" shape `TestResumeRejectedWhileClosing` asserts against, unreachable by that test because it lands the resume while the closing set is already marked (the 300 ms drain window), not inside the mark-vs-register gap. The window is microseconds wide, but it is the same class of defect the plan graded WR-03.
**Fix:** Make check-and-register atomic with the gate: hold `closingMu` across the final `isClosing` re-check and the `registry.Register` call (Register is in-memory and cheap), or have `CloseSession` re-`Get` the registry after marking the closing set and close whatever machine is registered under the same lock section, so a machine registered into a marked session is released by the teardown itself.

### WR-06: `cpAskRecorder` is unsynchronized against concurrent conn request handlers

**File:** `test/realdesktop/codingplan_test.go:327-377` (`recordAsk`, `recordHeaderAnswer`, `lastAskAt`, and the tap closure at 393-416)
**Issue:** The conn demux spawns one goroutine per request frame (stated in `internal/interact/responder.go:246` and implemented in `internal/zcode/conn.go`), so `conn.OnRequest` handlers run concurrently. The sibling recorder in the same package, `askRecorder` (`apicycle_test.go:754-766`), guards every field with a mutex and its comment states why: "Guarded by mu: request handlers are concurrent on the conn's read goroutine fan-out." The coding-plan variant dropped the mutex: `lastAsk`, `headerAsks`, `otherAsks`, and `otherAsksAt` are appended on conn handler goroutines and later read by the test goroutine (`cpToollessTurnLeg` iterates `asks.headerAsks` after `Prompt` returns) with no happens-before edge. That is a data race by inspection — it can produce torn reads, a misattributed answer shape (the "LAST pending entry absorbs the answer" index update races a concurrent ask's append), or a `-race` failure on the manual evidence run, i.e. an unreliable verdict on the phase's gated verification act.
**Fix:** Mirror the sibling: add `mu sync.Mutex` to `cpAskRecorder` and lock around `recordAsk`, `recordHeaderAnswer`, and `lastAskAt` (snapshot `headerAsks` out under the lock for the post-turn assertion).

## Info

### IN-01: `BuildAccount` reads the builtin table twice — revision and rules can pair from different generations

**File:** `internal/hostsvc/snapshot.go:189-196`
**Issue:** `BuildAccount` calls `readBuiltinRevision(install.BuiltinFile)` and then `readBuiltinProviderRules(install.BuiltinFile)` — two independent `os.ReadFile` passes over a live file a desktop update may rewrite between them. A mid-build update yields a push whose `basedOnZCodeBuiltinRevision` encodes revision N while `providers`/`states` mirror revision N+1's rules. Impact is bounded — the read-back want list is derived from the built snapshot itself (`wantModelsFromSnapshot`, auth.go), so the verdict stays self-consistent, and the stale `basedOn` string degrades honestly — but a single read (decode the file once, take revision and rules from the same bytes) removes the pairing hazard for free.
**Fix:** Read and unmarshal the file once into a combined struct (`{revision, config.providerConfigRules}`) and derive both `rev` and `rules` from that single decode.

### IN-02: `ResumeSession` adapter comment claims a validation the code does not perform

**File:** `internal/acpport/agent.go:434-444`
**Issue:** The doc comment says "the Handlers return value is validated non-empty (a front that failed loud already errored)", but the code discards the returned id entirely: `if _, err := a.h.ResumeSession(...)` — nothing checks it. The invariant does hold end-to-end, but via a different layer (`createSessionID` in the front fails loud on an empty id); the comment attributes the check to the wrong place, which will mislead the next maintainer who changes either layer.
**Fix:** Either correct the comment to state where the guarantee lives (the front's `createSessionID` dig-out), or actually validate: `sid, err := a.h.ResumeSession(...); if err == nil && sid == "" { return ..., acp.NewRequestError(CodeInternal, "acpport: resume returned an empty session id", nil) }`.

### IN-03: Keyring probe classifies a boundary-timed success as a timeout

**File:** `internal/interact/keyringprobe.go:224-240`
**Issue:** `classify` checks `ctx.Err() != nil` (parent cancellation) and then `pctx.Err() == context.DeadlineExceeded` BEFORE consulting the exec outcome. If the lookup subprocess completes successfully at the instant the probe's own bound expires, `pctx.Err()` is already set and the probe answers `ErrKeyringLookupTimeout` even though a valid keyring answer is in hand — a false negative at the deadline boundary (availability only; the next authenticate retries). Checking the successful outcome first (`execErr == nil && non-empty stdout` before the deadline arm) keeps the timeout class for genuinely hung lookups while honoring a result that beat the wire.
**Fix:** Reorder the switch: parent-ctx cancellation first, then the success arm, then the deadline arm, then the stderr-marker and default arms.

### IN-04: Impossible-state guard — nil re-initialization of an always-initialized map

**File:** `internal/acpfront/front.go:827-829`
**Category:** necessity
**Issue:** `recordSettings` guards with `if f.settings == nil { f.settings = make(map[string]json.RawMessage) }`, but `settings` is non-nil by construction: the only constructor, `New`, initializes it (`front.go:200: settings: make(map[string]json.RawMessage)`), the field is package-private and never reassigned anywhere in the package (verified by grep — the only other map assignment in the pair is `closing`, which IS lazily created because `New` never initializes it, so that guard is load-bearing while this one is not). The guarded state cannot occur in this stack.
**Fix:** Delete the nil check in `recordSettings` (keep `closing`'s lazy init — it is required). Optionally note the invariant in the struct-field comment.

---

_Reviewed: 2026-11-22T05:05:47Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
