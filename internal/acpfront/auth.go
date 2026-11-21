package acpfront

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
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

// errAPIKeyUnconfigured is the honest no-key arm (OQ4): the coded error
// names the env var — the one disposition that never fabricates, never
// falls back to reading the desktop's own config store.
var errAPIKeyUnconfigured = errors.New(
	"authenticate zai-api-key: no api key configured (set ZCODE_ACP_API_KEY or pass --api-key)")

// AuthConfig is the authenticate machinery the cmd wiring injects before
// any frame flows: the configured key binding (the D-05 config source),
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
}

// WithAuth injects the authenticate machinery (the plan 05-07 option on
// the front.go option set) before any frame flows.
func WithAuth(cfg AuthConfig) Option {
	return func(f *Front) { f.auth = &cfg }
}

// Authenticate runs one advertised auth method (the Handlers vocabulary).
// The method id validates against the advertised set; the zai-api-key arm
// runs the honest chain; the coding-plan arm answers its honest Phase-6
// disposition; an unknown id answers coded invalid-params. Every
// non-success is a coded error — never a fabricated success (PITFALL 3),
// never raw transport noise.
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

	if methodID != AuthMethodZaiAPIKey {
		// Honest disposition for the advertised-but-Phase-6 method: never a
		// fake success, never the misleading -32601 default (D-02).
		authLog(methodID, "declined (Phase 6)")

		return acpport.NewInternal(
			"authenticate %s: the OAuth coding-plan flow lands in Phase 6", methodID)
	}

	if err := f.authenticateAPIKey(ctx); err != nil {
		return err
	}

	authLog(methodID, "authenticated (read-back proven)")

	return nil
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
		"workspace": map[string]any{"workspacePath": dir, "workspaceKey": dir},
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
