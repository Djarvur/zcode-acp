package acpfront

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
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
}

// Option configures the Front at construction. (See front.go for the
// option set; WithAuth joins it with plan 05-07.)
func WithAuth(cfg AuthConfig) Option {
	return func(f *Front) { f.auth = &cfg }
}

// Authenticate runs one advertised auth method (the Handlers vocabulary).
// The method id validates against the advertised set; the zai-api-key arm
// runs the honest chain; the coding-plan arm answers its honest Phase-6
// disposition; an unknown id answers coded invalid-params.
//
// Skeleton posture (RED floor): every call answers the Phase-4 honest
// not-available error — never a fabricated success — while the composed
// chain lands in GREEN.
func (f *Front) Authenticate(_ context.Context, _ string) error {
	_ = f.auth

	return acpport.NewInternal(
		"authentication is not available in this build yet (z.ai auth flows land in Phases 5-6)")
}

// authenticateAPIKey runs the AUTH-01 chain for the zai-api-key arm. (The
// GREEN implementation composes hostsvc PushSnapshot → RefreshRegistry →
// the registry read-back with per-step coded error wrapping.)
func (f *Front) authenticateAPIKey(_ context.Context) error {
	return errAPIKeyUnconfigured
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
