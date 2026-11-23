package acpfront

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Front.Authenticate (plan 05-07, AUTH-01's composition tier): the honest
// authenticate{zai-api-key} chain — record the configured key → push the
// account snapshot (provider/updateAccountConfig) → refresh the registry
// (provider/refreshRegistry) → read the registry back (the PITFALL-1
// verdict) → only then the empty success. Every non-success answers a
// coded error naming the step that failed; a missing key answers the
// honest coded error naming ZCODE_ACP_API_KEY (OQ4 — never a fallback read
// of the desktop's config store, never a fabricated success); a cancelled
// authenticate records no success state and a re-authenticate is a clean
// retry (no half-authenticated state is observable).
//
// The authenticate carries NO credentials on the wire (ACP v1: {methodId}
// only): the key enters through the wrapper's own config (ZCODE_ACP_API_KEY
// / --api-key, D-05) as the injected KeySource — the same recorded key the
// cmd wiring feeds the responder's RuntimeHeadersKeySource and the snapshot
// builder (one key, one source, plan 05-07 Task 3).

// The advertised auth-method ids (the Initialize report's list — the
// single source both the advertisement and the authenticate validation
// read).
const (
	AuthMethodZaiAPIKey     = "zai-api-key"
	AuthMethodZaiCodingPlan = "zai-individual-coding-plan"
)

// AuthMode is the recorded authenticate outcome the runtime-headers source
// selects on (D-11, plan 06-05): the coding-plan arm records
// AuthModeCodingPlan at its success tail, the api-key arm AuthModeAPIKey —
// a cancelled or failed authenticate records NOTHING (no half-authenticated
// state is observable; the interruptibility contract). Round-3 arming: the
// coding-plan arm additionally ARMS the in-flight coding-plan value for the
// probe's materialization window (after the eager mint, before the poke —
// exactly the window the runtime can fire the runtime-headers ask in); the
// failure path RESTORES the prior value, so the observable post-call mode is
// unchanged — no recorded success means no mode change. The field lives on
// the Front (atomic — reads ride any goroutine; writes happen only inside
// the Authenticate dispatch and the coding-plan arm's armed window).
const (
	AuthModeNone       = ""
	AuthModeAPIKey     = "api-key"
	AuthModeCodingPlan = "coding-plan"
)

// authModeFor maps the advertised method id to its recorded mode value (the
// dispatch's single mapping point).
func authModeFor(methodID string) string {
	if methodID == AuthMethodZaiCodingPlan {
		return AuthModeCodingPlan
	}

	return AuthModeAPIKey
}

// AuthMode returns the recorded authenticate outcome (D-11): one of the
// AuthMode constants. The runtime-headers source (plan 06-07) selects its
// answerer on this value.
func (f *Front) AuthMode() string {
	if v, ok := f.authMode.Load().(string); ok {
		return v
	}

	return AuthModeNone
}

// recordAuthMode stores one successful authenticate's mode (the dispatch's
// single write point — arms never store).
func (f *Front) recordAuthMode(mode string) {
	f.authMode.Store(mode)
}

// errAPIKeyUnconfigured is the honest no-key arm (OQ4): the coded error
// names the env var — the one disposition that never fabricates, never
// falls back to reading the desktop's own config store.
var errAPIKeyUnconfigured = errors.New(
	"authenticate zai-api-key: no api key configured (set ZCODE_ACP_API_KEY or pass --api-key)")

// PatEnsurer is the consumer-side PAT mint seam (D-04, round-3 posture): the
// coding-plan authenticate arm calls Ensure EAGERLY — after the entitlement
// gate, BEFORE any wire traffic — arming the cache the probe's
// materialization ask answers from (the minter owns caching + single-flight,
// so this mint and any later ask-side Ensure dedupe into ONE mint). The
// verdict still gates the SUCCESS RECORD, never the mint: a failed or
// cancelled authenticate leaves no recorded mode and no fabricated success.
// Production wires the *hostsvc.PatMinter at cmd (plan 06-07); tests inject
// scripted doubles.
type PatEnsurer interface {
	// Ensure returns a current PAT, minting through the official chain when
	// the cache is stale or absent (the minter owns caching + single-flight).
	Ensure(ctx context.Context) (hostsvc.Pat, error)
}

// AuthConfig is the authenticate machinery the cmd wiring injects before
// any frame flow: the configured key binding (the D-05 config source),
// the resolved install (the snapshot's exact-revision builtin formula
// reads its zcode-builtin.json), and the read-back want list (the
// providerId/modelId identifiers the registry surfaces must carry — the
// PITFALL-1 verdict's expected models).
type AuthConfig struct {
	// Keys is the configured key source; nil, or a source reporting
	// unconfigured, answers errAPIKeyUnconfigured.
	Keys hostsvc.KeySource

	// Install is the resolved app-server install whose BuiltinFile feeds
	// the snapshot's basedOnZCodeBuiltinRevision formula.
	Install supervisor.Install

	// WantModels is the read-back want list — every entry must appear on a
	// registry surface before authenticate succeeds.
	WantModels []string

	// ReadbackDeadline bounds the registry read-back window (0 →
	// hostsvc.DefaultReadbackDeadline). Injectable per the established
	// deadline-knob posture: the window is authenticate policy, and tests
	// shorten it to exercise the coded read-back-negative arm without
	// racing the client's own context.
	ReadbackDeadline time.Duration

	// Creds is the desktop credential store (06-01) the coding-plan arm
	// reads its login from (D-12: the desktop login IS the credential —
	// no user-supplied wrapper config). Nil on api-key-only wirings.
	Creds *interact.CredentialStore

	// Pat is the PAT mint seam (D-04 round-3: minted EAGERLY at
	// authenticate — after the entitlement gate, before the push/probe —
	// arming the cache the probe's materialization ask answers from;
	// in-process only). Nil on api-key-only wirings; a nil seam on the
	// coding-plan path answers the coded internal error naming the missing
	// wiring.
	Pat PatEnsurer

	// Probe is the proactive keyring probe seam (D-02, OQ3 lock): run
	// BEFORE any snapshot traffic; a failure answers the coded actionable
	// error and nothing touches the wire. Nil on api-key-only wirings.
	Probe interact.KeyringProber

	// Entitle is the production entitlement gate (familysel.go wired as a
	// seam): the family decision the account overlay builder receives —
	// the gate may only omit families, never edit entries.
	Entitle func(ctx context.Context, family string) bool
}

// WithAuth injects the authenticate machinery (the plan 05-07 option on
// the front.go option set) before any frame flows.
func WithAuth(cfg AuthConfig) Option {
	return func(f *Front) { f.auth = &cfg }
}

// Authenticate runs one advertised auth method (the Handlers vocabulary).
// The method id validates against the advertised set; the zai-api-key arm
// runs the honest chain; the coding-plan arm composes the official chain
// (probe → login → account overlay → registry verdict → mint-once); an
// unknown id answers coded invalid-params. Every non-success is a coded
// error — never a fabricated success (PITFALL 3), never raw transport
// noise. A success records the arm's auth mode (D-11) as its LAST step; a
// cancelled or failed authenticate records nothing, so a re-authenticate is
// a clean retry.
func (f *Front) Authenticate(ctx context.Context, methodID string) error {
	advertised := advertisedAuthMethods()

	ids := make([]string, 0, len(advertised))
	for _, m := range advertised {
		ids = append(ids, m.ID)
	}

	if !slices.Contains(ids, methodID) {
		return acpport.NewInvalidParams(
			"authenticate: unknown methodId %q (advertised: %s)", methodID, strings.Join(ids, ", "))
	}

	var err error

	switch methodID {
	case AuthMethodZaiCodingPlan:
		err = f.authenticateCodingPlan(ctx)
	default:
		err = f.authenticateAPIKey(ctx)
	}

	if err != nil {
		return err
	}

	f.recordAuthMode(authModeFor(methodID))

	authLog(methodID, "authenticated (read-back proven)")

	return nil
}

// authCodingPlanFamily is the provider family the coding-plan arm
// authenticates (the literal at the production call site; the entitlement
// gate and the store reader stay family-generic).
const authCodingPlanFamily = "zai"

// authenticateCodingPlan composes the official individual coding-plan chain
// (AUTH-02, plan 06-05; 06-RESEARCH.md's composition sketch; round-3 order)
// at the front tier — every step a coded error naming the step, the verdict
// ALWAYS the registry read-back (PITFALL 1: the received ack is never the
// verdict):
//
//  1. the PROACTIVE keyring probe (D-02, AUTH-04) — BEFORE any snapshot
//     traffic; a failure answers the coded actionable error naming the
//     missing piece and nothing touches the wire;
//  2. the desktop login read (D-03/D-12) through the 06-01 credential
//     store — a missing login answers the coded desktop-first error before
//     any traffic (T-06-16);
//  3. the entitlement gate wiring check (the gate itself runs inside the
//     overlay build below; it may only omit families, never edit entries);
//  4. the EAGER PAT mint (D-04 round-3) — after the entitlement gate,
//     BEFORE any wire traffic: the cache is armed while the chain is still
//     pre-traffic, so the materialization ask the probe's poke can fire
//     answers instantly (single-flight dedupes with any later Ensure);
//  5. the account overlay build (Pattern 2) through BuildAccount with the
//     production entitlement gate;
//  6. the want list derived from the BUILT snapshot's providers
//     (providerId/modelId over each entitled entry's builtinModelIds —
//     D-08's verdict-side twin: the want list mirrors what we push, never a
//     hardcoded catalog);
//  7. the read-back watcher registered BEFORE the push traffic (the conn
//     hook contract), then push → refresh → the probe (create → ready → the
//     deterministic setModel poke → close — EXP-CODINGPLAN-CYCLE-01 round-3
//     root cause: create+close alone never materializes the account
//     provider, so no runtime-headers ask fires and the account models
//     never publish) → Await — all under the ARMED in-flight mode, restored
//     on any failure;
//  8. the auth mode record happens in the dispatch after this arm returns
//     nil (D-11) — the armed window's value already reads coding-plan; the
//     record is idempotent.
func (f *Front) authenticateCodingPlan(ctx context.Context) error {
	if f.auth == nil {
		return acpport.NewInternal("authenticate: auth machinery unconfigured")
	}

	// 1+2. The probe (D-02) and the desktop login read (D-03/D-12) gate
	// every wire step — both answer coded actionable errors before any
	// traffic.
	if _, err := f.codingPlanLogin(ctx); err != nil {
		return err
	}

	// 3. The entitlement gate wiring check — a missing gate is the coded
	// missing-wiring posture before anything else runs.
	if f.auth.Entitle == nil {
		return acpport.NewInternal("authenticate: entitlement gate unconfigured (no familysel gate wired)")
	}

	// 4. D-04 round-3: the EAGER mint — the cache arms while the chain is
	// still pre-traffic; a mint failure is a coded pre-traffic error (no
	// push, no refresh, no probe session ever spawns).
	if f.auth.Pat == nil {
		return acpport.NewInternal("authenticate: PAT mint unconfigured (no PatEnsurer wired)")
	}

	if _, err := f.auth.Pat.Ensure(ctx); err != nil {
		return acpport.NewInternal("authenticate: PAT mint: %v", err)
	}

	// 5. The account overlay (Pattern 2) — the production entitlement gate
	// wired as the caller-supplied decision (06-03's BuildAccount
	// contract); a rejected family is omitted, never edited.
	snap, err := hostsvc.NewSnapshotBuilder(nil).BuildAccount(f.auth.Install,
		func(family string) bool { return f.auth.Entitle(ctx, family) })
	if err != nil {
		return acpport.NewInternal("authenticate: build snapshot: %v", err)
	}

	// 6. The want list mirrors the built overlay (D-08's verdict-side twin).
	want := wantModelsFromSnapshot(snap)

	// 7. The read-back setup — identical to the api-key arm: the watcher
	// registers BEFORE the push traffic.
	watchOpts := []hostsvc.WatchOption{}
	if f.auth.ReadbackDeadline > 0 {
		watchOpts = append(watchOpts, hostsvc.WithReadbackDeadline(f.auth.ReadbackDeadline))
	}

	watcher, remove := hostsvc.WatchRegistry(f.conn, want, watchOpts...)
	defer remove()

	if _, err := hostsvc.PushEnvelope(ctx, f.conn, snap.Params); err != nil {
		return acpport.NewInternal("authenticate: push snapshot: %v", err)
	}

	if err := hostsvc.RefreshRegistry(ctx, f.conn); err != nil {
		return acpport.NewInternal("authenticate: refresh registry: %v", err)
	}

	// The in-flight arming (D-11 round-3): the mode reads coding-plan for
	// the probe's materialization window — the runtime-headers ask the poke
	// can fire is served by the mode-aware source only while the mode reads
	// coding-plan. A failure past this point RESTORES the prior value: a
	// failed or cancelled authenticate still records nothing new.
	priorMode := f.armAuthMode()

	if !watcher.Applied() {
		f.probeCreateSetModelReadback(ctx, watcher, want)
	}

	// The verdict is the registry's (PITFALL 1): the received ack is
	// receipt evidence only, and the poke's own outcome is evidence, never
	// the verdict. A cancelled or expired wait surfaces the coded read-back
	// error — the armed window closes, no success state.
	if _, err := watcher.Await(ctx); err != nil {
		f.restoreAuthMode(priorMode)

		return acpport.NewInternal("authenticate: registry read-back: %v", err)
	}

	return nil
}

// armAuthMode arms the in-flight coding-plan mode (the probe's
// materialization window — after the eager mint, before the poke) and
// returns the mode it replaced; restoreAuthMode puts that prior value back
// on the arm's failure paths, so the observable post-call mode is exactly
// what it was before the attempt (D-11's recorded-nothing contract, round-3
// arming posture).
func (f *Front) armAuthMode() string {
	prior := f.AuthMode()
	f.authMode.Store(AuthModeCodingPlan)

	return prior
}

// restoreAuthMode closes the armed window (see armAuthMode).
func (f *Front) restoreAuthMode(mode string) {
	f.authMode.Store(mode)
}

// codingPlanLogin runs the arm's pre-traffic prologue: the PROACTIVE keyring
// probe (D-02, AUTH-04 — BEFORE any snapshot traffic; its coded actionable
// errors wrap verbatim, raw exec noise never passes) then the desktop login
// read (D-03/D-12) through the 06-01 credential store — a missing login
// answers the coded desktop-first error before any traffic (T-06-16). The
// Bearer-stripped login value lives in this call's scope only (the mint
// seam re-reads its own credential; the value never reaches a log).
func (f *Front) codingPlanLogin(ctx context.Context) (string, error) {
	if f.auth.Probe == nil {
		return "", acpport.NewInternal("authenticate: keyring probe unconfigured (no production prober wired)")
	}

	if err := f.auth.Probe(ctx); err != nil {
		return "", acpport.NewInternal("authenticate: keyring unavailable: %v", err)
	}

	if f.auth.Creds == nil {
		return "", acpport.NewInternal(
			"authenticate: no desktop login (log in via the ZCode desktop first)")
	}

	login, ok := f.auth.Creds.LoginToken(ctx, authCodingPlanFamily)
	if !ok || login == "" {
		return "", acpport.NewInternal(
			"authenticate: no desktop login (log in via the ZCode desktop first)")
	}

	return login, nil
}

// wantModelsFromSnapshot derives the read-back want list from the BUILT
// account snapshot: providerId/modelId over each entitled entry's
// builtinModelIds, sorted for deterministic watcher matching. This is the
// D-08 verdict-side twin — the want list mirrors what we push, never a
// hardcoded catalog; a malformed entry degrades to omission (the build
// already validated the table).
func wantModelsFromSnapshot(snap hostsvc.Snapshot) []string {
	providers, _ := snap.Params["providers"].(map[string]any)

	want := make([]string, 0, len(providers))
	for providerID, entry := range providers {
		rule, _ := entry.(map[string]any)
		modelIDs, _ := rule["builtinModelIds"].([]string)

		for _, modelID := range modelIDs {
			want = append(want, providerID+"/"+modelID)
		}
	}

	sort.Strings(want)

	return want
}

// authenticateAPIKey runs the AUTH-01 chain: the configured-key gate → the
// read-back watcher registration (BEFORE the push traffic — the conn hook
// contract; a registry surface emitted between the refresh reply and a
// late registration would be missed) → snapshot build → push → refresh →
// read-back verdict, each step wrapping its failure into a coded error
// naming the step (transport noise never passes raw).
func (f *Front) authenticateAPIKey(ctx context.Context) error {
	if f.auth == nil || f.auth.Keys == nil {
		return acpport.NewInternal("%v", errAPIKeyUnconfigured)
	}

	if apiKey, _, ok := f.auth.Keys.APIKey(ctx); !ok || apiKey == "" {
		// The sentinel wraps through the seam vocabulary so the wire
		// message carries the env var name verbatim (the client's readable
		// surface — never the generic internal mapping's data-only shape).
		return acpport.NewInternal("%v", errAPIKeyUnconfigured)
	}

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

	// The field-verified read-back arm (plan 05-11, EXP-READBACK-DIAG-01):
	// real-desktop 3.18.2 emits NO state.updated patch for push+refresh —
	// the verdict surface is the first create reply's settings.model.available
	// (dialect §8). When a patch already latched (a desktop that does emit
	// one), the probe is unnecessary — the observer stays for patches the
	// desktop does emit. Either way the probe is evidence, never the verdict:
	// the verdict remains the watcher's Await below.
	if !watcher.Applied() {
		f.probeCreateReadback(ctx, watcher)
	}

	// The verdict is the registry's (PITFALL 1): the received ack is
	// receipt evidence only. A cancelled or expired wait surfaces the
	// coded read-back error — no success state is recorded, so a
	// re-authenticate is a clean retry (the interruptibility contract).
	if _, err := watcher.Await(ctx); err != nil {
		return acpport.NewInternal("authenticate: registry read-back: %v", err)
	}

	return nil
}

// probeCreateReadback offers the field-verified registry surface to the
// watcher (plan 05-11, the branch-(c) correction): one raw probe
// session/create over a temp-dir cwd — NO model binding, NO turn
// (registry-level ops spend nothing) — whose reply feeds
// watcher.OfferCreateReply, then a raw session/close for the probe id
// (best-effort; the field-verified {closed:true} shape answers it). The
// probe id digs out through createSessionID READ-ONLY (front.go stays a
// same-wave sibling's file). Every probe failure is recorded structurally
// and skips its step: a refused create or a reply without the want models
// leaves the verdict to Await — the ack and the probe's transport success
// are never the verdict, and a cancelled ctx surfaces through the same
// coded read-back error with no success recorded (the interruptibility
// contract).
func (f *Front) probeCreateReadback(ctx context.Context, watcher *hostsvc.RegistryWatcher) {
	dir, err := os.MkdirTemp("", "zcode-acp-auth-probe-")
	if err != nil {
		slog.Warn("acpfront: authenticate probe create skipped (temp dir)", "err", err)

		return
	}

	//nolint:errcheck // best-effort probe cleanup; a stale temp dir must never fail the auth chain
	defer os.RemoveAll(dir)

	res, err := f.conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		wireKeyWorkspace: map[string]any{wireKeyWorkspacePath: dir, wireKeyWorkspaceKey: dir},
	})
	if err != nil {
		slog.Warn("acpfront: authenticate probe create refused "+
			"(the registry verdict still awaits its surfaces)", "err", err)

		return
	}

	// The reply IS the read-back surface — offer it BEFORE the id dig so a
	// reply lacking a session id still latches.
	watcher.OfferCreateReply(res)

	sid, err := createSessionID(res, "session/create (authenticate probe)")
	if err != nil {
		slog.Warn("acpfront: authenticate probe reply carried no session id (close skipped)", "err", err)

		return
	}

	if _, err := f.conn.Call(ctx, zcode.MethodSessionClose, map[string]any{
		wireKeySessionID: sid,
	}); err != nil {
		slog.Warn("acpfront: backend probe session/close failed (auth continues)", "session", sid, "err", err)
	}
}

// probeAccountProvider is the coding-plan account family the poke prefers —
// the providerId the builtin table's zhipu-account rule carries for the
// individual coding plan (the family the arm authenticates and the overlay
// entitles).
const probeAccountProvider = "account:zai-individual-coding-plan"

// probeFlashSuffix is the modelId suffix the deterministic pick prefers (the
// field's fast-tier model of the family).
const probeFlashSuffix = "-Flash"

// ProbeWantModel picks the coding-plan probe's deterministic setModel target
// out of the read-back want list (providerId/modelId entries): the
// coding-plan account family's entries when the want list carries any, within
// them the first modelId ending in -Flash, else the family's first; an empty
// family selection falls back to the first well-formed want entry overall
// (the entitlement gate may have omitted the coding-plan family — the poke
// still binds something deterministic); an empty (or all-malformed) want list
// answers "" — the poke is skipped, nothing honest to bind. Pure selection
// over the caller's order (the want list is sorted at derivation): no map
// iteration, deterministic by construction.
func ProbeWantModel(want []string) string {
	family := make([]string, 0, len(want))

	for _, entry := range want {
		if pid, _, cut := strings.Cut(entry, "/"); cut && pid == probeAccountProvider {
			family = append(family, entry)
		}
	}

	pool := family
	if len(pool) == 0 {
		pool = make([]string, 0, len(want))

		for _, entry := range want {
			if _, _, cut := strings.Cut(entry, "/"); cut {
				pool = append(pool, entry)
			}
		}
	}

	for _, entry := range pool {
		if _, modelID, _ := strings.Cut(entry, "/"); strings.HasSuffix(modelID, probeFlashSuffix) {
			return entry
		}
	}

	if len(pool) == 0 {
		return ""
	}

	return pool[0]
}

// probeCreateSetModelReadback is the coding-plan probe (probeCreateReadback's
// successor, EXP-CODINGPLAN-CYCLE-01 round-3 root cause): one raw probe
// session/create over a temp-dir cwd — NO turn (registry-level ops spend
// nothing) — whose reply feeds watcher.OfferCreateReply; the create reply IS
// the ready gate (the confirmed snapshot returns once the runtime
// materialized the session), then the deterministic setModel POKE with
// persistAsWorkspaceLastUsed:false and NO options object (the probe session
// has no confirmed reasoning level — the omit-shape): the model-selection
// materialization that makes the account provider publish. Without the poke
// the runtime never materializes the account provider — no
// interaction/requestProviderRuntimeHeaders ask fires, no PAT answer flows,
// and the account models never reach any registry surface (attempts 1-3).
// BOTH poke outcomes are handled without ever fabricating the verdict: an ok
// poke leaves the verdict to the watcher (a subsequent
// interaction/requestProviderRuntimeHeaders ask is answered by the mode-aware
// source from the cache the eager mint armed — the window runs under the
// armed in-flight mode); a coded poke refusal is recorded structurally as
// probe evidence and the verdict REMAINS the watcher's Await. A raw
// session/close for the probe id trails the poke either way (best-effort;
// the field-verified {closed:true} shape answers it). The probe id digs out
// through createSessionID READ-ONLY (front.go stays a same-wave sibling's
// file). Every probe failure is recorded structurally and skips its step; a
// cancelled ctx surfaces through the same coded read-back error with no
// success recorded (the interruptibility contract).
func (f *Front) probeCreateSetModelReadback(ctx context.Context, watcher *hostsvc.RegistryWatcher, want []string) {
	dir, err := os.MkdirTemp("", "zcode-acp-auth-probe-")
	if err != nil {
		slog.Warn("acpfront: authenticate probe create skipped (temp dir)", "err", err)

		return
	}

	//nolint:errcheck // best-effort probe cleanup; a stale temp dir must never fail the auth chain
	defer os.RemoveAll(dir)

	res, err := f.conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		wireKeyWorkspace: map[string]any{wireKeyWorkspacePath: dir, wireKeyWorkspaceKey: dir},
	})
	if err != nil {
		slog.Warn("acpfront: authenticate probe create refused "+
			"(the registry verdict still awaits its surfaces)", "err", err)

		return
	}

	// The reply IS the read-back surface — offer it BEFORE the id dig so a
	// reply lacking a session id still latches.
	watcher.OfferCreateReply(res)

	sid, err := createSessionID(res, "session/create (authenticate probe)")
	if err != nil {
		slog.Warn("acpfront: authenticate probe reply carried no session id (poke and close skipped)", "err", err)

		return
	}

	// The poke: the deterministic want model — the account provider's
	// materializer (the model selection the runtime's
	// modelSelectionReadinessSource machinery reacts to). Account models
	// are LEVEL-STRICT: the poke carries the want model's own declared
	// level (its DefaultLevel, else its first reasoning level — the
	// create reply's settings snapshot lists both; the 2026-11-23 field
	// refusal "Reasoning level is required" proved the omit-shape fatal,
	// and an unmaterialized account provider shrinks the /model catalog
	// to the builtin scope).
	// The poke: the deterministic want model — the account provider's
	// materializer. Account models are LEVEL-STRICT: the poke carries the
	// model's own declared level (the 2026-11-23 field refusal "Reasoning
	// level is required" proved the omit-shape fatal).
	if model := ProbeWantModel(want); model != "" {
		f.probePokeSetModel(ctx, sid, res, model)
	} else {
		slog.Warn("acpfront: authenticate probe setModel skipped (the want list carried no bindable model)")
	}

	if _, err := f.conn.Call(ctx, zcode.MethodSessionClose, map[string]any{
		wireKeySessionID: sid,
	}); err != nil {
		slog.Warn("acpfront: backend probe session/close failed (auth continues)", "session", sid, "err", err)
	}
}

// advertisedAuthMethods returns the advertised auth-method definitions the
// Initialize report carries — the single source Initialize and the
// authenticate validation share.
func advertisedAuthMethods() []acpport.AuthMethodDef {
	return []acpport.AuthMethodDef{
		{ID: AuthMethodZaiAPIKey, Name: "z.ai API key"},
		{ID: AuthMethodZaiCodingPlan, Name: "z.ai Individual Coding Plan"},
	}
}

// authLog records one authenticate outcome structurally (method + outcome
// — never key material).
func authLog(methodID, outcome string) {
	slog.Info("acpfront: authenticate", "method", methodID, "outcome", outcome)
}

// probePokeSetModel fires the probe session's setModel poke with the want
// model — carrying the model's OWN declared reasoning level (its
// DefaultLevel, else its first reasoning level: the create reply's settings
// snapshot lists both; the 2026-11-23 field refusal "Reasoning level is
// required" proved the omit-shape fatal for account models).
func (f *Front) probePokeSetModel(ctx context.Context, sid string, res json.RawMessage, model string) {
	providerID, modelID, _ := strings.Cut(model, "/")

	modelPayload := map[string]any{
		wireKeyProviderID: providerID,
		wireKeyModelID:    modelID,
	}

	if settings, ok := ParseSettings(res); ok {
		for i := range settings.Model.Available {
			entry := &settings.Model.Available[i]

			if entry.Ref.ProviderID != providerID || entry.Ref.ModelID != modelID {
				continue
			}

			level := entry.DefaultLevel
			if level == "" && len(entry.ReasoningLevels) > 0 {
				level = entry.ReasoningLevels[0].Value
			}

			if level != "" {
				modelPayload["options"] = map[string]any{wireKeyReasoningLevel: level}
			}

			break
		}
	}

	if _, err := f.conn.Call(ctx, zcode.MethodSessionSetModel, map[string]any{
		wireKeySessionID:       sid,
		optionIDModel:          modelPayload,
		wireKeyPersistLastUsed: false,
	}); err != nil {
		slog.Warn("acpfront: authenticate probe setModel refused "+
			"(coded evidence recorded; the verdict still awaits its surfaces)", "model", model, "err", err)
	}
}
