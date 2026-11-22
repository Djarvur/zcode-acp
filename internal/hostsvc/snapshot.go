// Package hostsvc implements AUTH-01's execution tier (plan 05-06): the
// api-key snapshot push (provider/updateAccountConfig with the
// exact-revision builtin formula over the field-verified EMPTY personal-layer
// overlay), the provider/refreshRegistry call, the registry read-back helper
// (PITFALL 1 discipline — the ack is never the verdict), and the
// runtime-headers answer over the 05-01 KeySource seam (D-02's success
// shape). The front's authenticate (plan 05-07) composes these step
// functions.
package hostsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// revisionPrefix is the stable form of the wrapper-generated snapshot
// revision (reference-adapter precedent "account:bridge:<timestamp>",
// docs/research/reference-adapters.md row 3): any non-empty string is
// schema-valid, so a testable deterministic form is the honest choice.
const revisionPrefix = "account-bridge:"

// refreshReason is the provider/refreshRegistry params reason (the probe's
// field-verified params shape {reason} — EXP-WORKSPACEKEY-01 answered
// {"refreshed":true}).
const refreshReason = "account-config-sync"

// zhipuAccountAccess is the exact access-type string the account overlay
// filters the builtin table's rules by (RESEARCH Pattern 2: the
// config.providerConfigRules.providerRules[] rows whose config.access.type
// is zhipu-account are the account providers).
const zhipuAccountAccess = "zhipu-account"

// The coded account-build error classes (static sentinels — the variable
// detail rides fmt.Errorf %w, never a bare dynamic error; the messages keep
// the capture-verbatim wire-key spellings).
var (
	errNoRulesToEntitle = errors.New("provider rules to entitle after the entitlement gate — " +
		"an account push with nothing to entitle is a misconfiguration, " +
		"never an empty overlay silently pushed on the coding-plan path")
	errMissingProviderRules   = errors.New("missing config.providerConfigRules.providerRules")
	errMissingProviderID      = errors.New("missing providerId")
	errMissingAccessBlock     = errors.New("missing config.access block")
	errMissingAccessMode      = errors.New("missing config.access mode")
	errMissingBuiltinModelIDs = errors.New("missing config.builtinModelIds")
)

// defaultRevision renders the push revision: account-bridge:<unix-nano> —
// fresh per build, stable in form.
func defaultRevision() string {
	return fmt.Sprintf("%s%d", revisionPrefix, time.Now().UnixNano())
}

// KeySource supplies the authenticate-sourced api-key material (the 05-01
// responder seam shape: apiKey/apiKeyID accessors). Implementations hold
// the material in memory only — never in logs, captures, or fixtures
// (T-5-15).
type KeySource interface {
	// APIKey returns the configured key material; ok=false when
	// unconfigured (the honest coded-error arm).
	APIKey(ctx context.Context) (apiKey, apiKeyID string, ok bool)
}

// StaticKey is the config-sourced KeySource: material handed over at
// authenticate time from the wrapper's own config (env ZCODE_ACP_API_KEY /
// flag per OQ4), held in memory only.
type StaticKey struct {
	apiKey string
	keyID  string
}

// NewStaticKey returns a KeySource over fixed material (either part may be
// empty — an empty key pair answers the unconfigured arm).
func NewStaticKey(apiKey, keyID string) *StaticKey {
	return &StaticKey{apiKey: apiKey, keyID: keyID}
}

// APIKey returns the fixed material.
func (s *StaticKey) APIKey(_ context.Context) (apiKey, keyID string, ok bool) {
	return s.apiKey, s.keyID, s.apiKey != ""
}

// SnapshotOption configures a SnapshotBuilder at construction.
type SnapshotOption func(*SnapshotBuilder)

// WithRevisionSource overrides the push-revision generator (tests and
// corpus replays inject a fixed value for byte-deterministic goldens; the
// default generates a fresh account-bridge:<unix-nano> per build). The
// builtin-revision formula is NOT injectable — it is always computed at
// runtime via supervisor.BuiltinRevision over the resolved path
// (T-5-16: never hard-coded, never cached).
func WithRevisionSource(source func() string) SnapshotOption {
	return func(b *SnapshotBuilder) { b.revision = source }
}

// SnapshotBuilder builds the provider/updateAccountConfig snapshot params:
// the strict envelope {revision, basedOnZCodeBuiltinRevision, providers,
// states} whose overlay is intentionally EMPTY for the api-key path (zai-api
// is the personal layer, untouched by account snapshots — field-verified
// no-op; the protocol comment places keys/JWTs/dynamic headers in the
// request-time runtime-headers protocol, never the snapshot).
type SnapshotBuilder struct {
	keys     KeySource
	revision func() string
}

// NewSnapshotBuilder returns a builder over the account-credential binding
// (keys — the same binding the runtime-headers answer serves; the envelope
// itself carries no key material) with options applied.
func NewSnapshotBuilder(keys KeySource, opts ...SnapshotOption) *SnapshotBuilder {
	b := &SnapshotBuilder{keys: keys, revision: defaultRevision}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Snapshot is one built push envelope and its identity: Revision is the
// wrapper-generated snapshot revision, BasedOn the exact builtin-revision
// formula output, Params the strict envelope to push.
type Snapshot struct {
	Revision string
	BasedOn  string
	Params   map[string]any
}

// Build computes the strict envelope for install, reading the recorded
// revision int out of the active zcode-builtin.json (the probe-time read
// the realdesktop tier records — a live desktop may update mid-probe; the
// value is read, never assumed).
func (b *SnapshotBuilder) Build(install supervisor.Install) (Snapshot, error) {
	rev, err := readBuiltinRevision(install.BuiltinFile)
	if err != nil {
		return Snapshot{}, fmt.Errorf("hostsvc: snapshot: %w", err)
	}

	return b.BuildRevision(install, rev), nil
}

// BuildRevision computes the strict envelope over an explicitly supplied
// recorded revision int (probe-time reads hold it already; the corpus
// fixture passes a fixed synthetic value). basedOnZCodeBuiltinRevision is
// the EXACT zcode-builtin:<rev>:<sha256(resolved-path)> formula REUSED from
// supervisor.BuiltinRevision — never re-derived, never hard-coded.
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

// BuildAccount builds the coding-plan ACCOUNT snapshot (AUTH-02, RESEARCH
// Pattern 2 — the NON-empty overlay, the exact opposite shape of
// BuildRevision's intentional empty one): providers carries one entry per
// builtin-table rule whose config.access.type is zhipu-account, each
// {builtinModelIds: verbatim from the table, access {type, accountType,
// mode, entitled true}} — the COMPLETE access schema: the runtime registry
// validates each pushed provider against the strict completeZhipuAccount
// access schema and silently EXCLUDES a provider with validation issues
// (the EXP-CODINGPLAN-CYCLE-01 attempt-4 symptom: "Provider Registry 中不
// 存在 Model" with state.updated never carrying the account models), so
// accountType + mode are propagated verbatim from the table rule;
// states carries per provider {availability available, entitled true,
// current true}. The rules come from the desktop's OWN bundled table — the
// SAME install.BuiltinFile readBuiltinRevision opens (D-08's snapshot-side
// twin: verbatim mirror, no renaming/reordering/model-list editing;
// T-06-09). Entitlement is a decision the builder RECEIVES: entitled is the
// caller's gate (production: setting.json selections/modes + store oauth
// token, authored in 06-05) and a rejected family is omitted from providers
// AND states entirely — the gate may only omit, never edit. A malformed
// table answers a coded error — never an empty overlay silently pushed as
// if it were the account config.
//
// basedOnZCodeBuiltinRevision stays the EXACT supervisor.BuiltinRevision
// formula over the same resolved install path (T-06-08 / the Phase-5 lock).
func (b *SnapshotBuilder) BuildAccount(
	install supervisor.Install,
	entitled func(family string) bool,
) (Snapshot, error) {
	rev, err := readBuiltinRevision(install.BuiltinFile)
	if err != nil {
		return Snapshot{}, fmt.Errorf("hostsvc: snapshot: %w", err)
	}

	rules, err := readBuiltinProviderRules(install.BuiltinFile)
	if err != nil {
		return Snapshot{}, fmt.Errorf("hostsvc: snapshot: %w", err)
	}

	providers := make(map[string]any, len(rules))
	states := make(map[string]any, len(rules))

	for _, rule := range rules {
		if rule.AccessType != zhipuAccountAccess {
			continue // only zhipu-account rules are account providers (apiKey/personal-layer rows stay out)
		}

		if !entitled(rule.Family) {
			continue // the gate may only omit families — never edit entries (T-06-09)
		}

		// access is the table's own access block mirrored with entitled
		// true — every field capture-verbatim, the COMPLETE schema the
		// runtime registry validates (missing fields = silent registry drop).
		providers[rule.ProviderID] = map[string]any{
			"builtinModelIds": rule.ModelIDs,
			"access": map[string]any{
				"type":        rule.AccessType,
				"accountType": rule.Family,
				"mode":        rule.AccessMode,
				"entitled":    true,
			},
		}

		states[rule.ProviderID] = map[string]any{
			"availability": "available",
			"entitled":     true,
			"current":      true, // required for entitled zhipu-account entries (RESEARCH Pattern 2)
		}
	}

	basedOn := supervisor.BuiltinRevision(install.BuiltinFile, rev)
	revision := b.revision()

	if len(providers) == 0 {
		return Snapshot{}, fmt.Errorf("hostsvc: snapshot: builtin table %s: no %s %w",
			install.BuiltinFile, zhipuAccountAccess, errNoRulesToEntitle)
	}

	params := map[string]any{
		"revision":                    revision,
		"basedOnZCodeBuiltinRevision": basedOn,
		"providers":                   providers,
		"states":                      states,
	}

	return Snapshot{Revision: revision, BasedOn: basedOn, Params: params}, nil
}

// builtinProviderRule is one account-provider rule dug out of the desktop's
// bundled builtin table (config.providerConfigRules.providerRules[]) —
// capture-verbatim camelCase wire fields: the wrapper mirrors exactly these,
// never edits them (T-06-09).
type builtinProviderRule struct {
	ProviderID string   // providerId — the providers/states map key
	ModelIDs   []string // config.builtinModelIds — carried verbatim
	AccessType string   // config.access.type — the zhipu-account filter
	Family     string   // config.access.accountType — the entitlement-gate family + pushed access
	AccessMode string   // config.access.mode — the complete-access-schema mode, pushed verbatim
}

// readBuiltinProviderRules digs the account-provider rules out of the SAME
// zcode-builtin.json readBuiltinRevision opens. Capture-verbatim camelCase
// discipline — but UNLIKE the ack dig-outs, undecodable here is a coded
// error, NOT absent: a malformed table (undecodable JSON, missing
// providerRules, a rule missing its access block) must fail the build, never
// push a degraded overlay.
func readBuiltinProviderRules(path string) ([]builtinProviderRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read builtin provider rules: %w", err)
	}

	//nolint:tagliatelle // wire shape: the desktop's builtin-table keys are capture-verbatim camelCase
	var parsed struct {
		Config struct {
			ProviderConfigRules struct {
				ProviderRules []json.RawMessage `json:"providerRules"`
			} `json:"providerConfigRules"`
		} `json:"config"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("builtin table %s: parse: %w", path, err)
	}

	rawRules := parsed.Config.ProviderConfigRules.ProviderRules
	if rawRules == nil {
		return nil, fmt.Errorf("builtin table %s: %w", path, errMissingProviderRules)
	}

	rules := make([]builtinProviderRule, 0, len(rawRules))

	for i, raw := range rawRules {
		rule, err := digBuiltinRule(path, i, raw)
		if err != nil {
			return nil, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// digBuiltinRule decodes one raw provider rule out of the builtin table and
// locks its required fields — a rule missing its providerId, its access
// block, its model list, or (for a zhipu-account row) its access mode fails
// the build with the same coded class as the table-level checks
// (capture-verbatim camelCase wire fields, never renamed). The mode lock is
// the completeness requirement: the runtime registry validates each pushed
// zhipu-account provider against the COMPLETE access schema and silently
// excludes a provider with validation issues (EXP-CODINGPLAN-CYCLE-01
// attempt-4), so a mode-less rule can never mirror into a valid push.
func digBuiltinRule(path string, i int, raw json.RawMessage) (builtinProviderRule, error) {
	//nolint:tagliatelle // wire shape: the desktop's builtin-table keys are capture-verbatim camelCase
	var rule struct {
		ProviderID string `json:"providerId"`
		Config     struct {
			BuiltinModelIDs []string `json:"builtinModelIds"`
			Access          *struct {
				Type        string `json:"type"`
				AccountType string `json:"accountType"`
				Mode        string `json:"mode"`
			} `json:"access"`
		} `json:"config"`
	}

	if err := json.Unmarshal(raw, &rule); err != nil {
		return builtinProviderRule{}, fmt.Errorf("builtin table %s: provider rule %d: decode: %w", path, i, err)
	}

	if rule.ProviderID == "" {
		return builtinProviderRule{}, fmt.Errorf("builtin table %s: provider rule %d: %w",
			path, i, errMissingProviderID)
	}

	if rule.Config.Access == nil {
		return builtinProviderRule{}, fmt.Errorf("builtin table %s: provider rule %d (%s): %w",
			path, i, rule.ProviderID, errMissingAccessBlock)
	}

	if rule.Config.Access.Type == zhipuAccountAccess && rule.Config.Access.Mode == "" {
		return builtinProviderRule{}, fmt.Errorf("builtin table %s: provider rule %d (%s): %w",
			path, i, rule.ProviderID, errMissingAccessMode)
	}

	if rule.Config.BuiltinModelIDs == nil {
		return builtinProviderRule{}, fmt.Errorf("builtin table %s: provider rule %d (%s): %w",
			path, i, rule.ProviderID, errMissingBuiltinModelIDs)
	}

	return builtinProviderRule{
		ProviderID: rule.ProviderID,
		ModelIDs:   rule.Config.BuiltinModelIDs,
		AccessType: rule.Config.Access.Type,
		Family:     rule.Config.Access.AccountType,
		AccessMode: rule.Config.Access.Mode,
	}, nil
}

// readBuiltinRevision reads the recorded revision number out of the active
// zcode-builtin.json.
func readBuiltinRevision(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read builtin revision: %w", err)
	}

	var parsed struct {
		Revision int `json:"revision"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, fmt.Errorf("parse builtin revision %s: %w", path, err)
	}

	return parsed.Revision, nil
}

// Ack is the provider/updateAccountConfig reply (dialect §8) — receipt
// evidence only. The schema's own comment warns the ack is not application
// ("收到账号结果不代表配套 Built-in 已到达"): success is proven ONLY by the
// registry read-back (PITFALL 1), never by this value.
type Ack struct {
	ReceivedRevision string
	ProviderCount    int
	Status           string // "received" | "unchanged"
}

// PushSnapshot builds with the default builder and runs the
// provider/updateAccountConfig Call.
func PushSnapshot(ctx context.Context, conn *zcode.Conn, install supervisor.Install) (Ack, error) {
	snap, err := NewSnapshotBuilder(nil).Build(install)
	if err != nil {
		return Ack{}, err
	}

	return PushEnvelope(ctx, conn, snap.Params)
}

// PushEnvelope runs the provider/updateAccountConfig Call with params —
// the Call + wrap-on-error pattern (front.go NewSession discipline) — and
// parses the receipt ack.
func PushEnvelope(ctx context.Context, conn *zcode.Conn, params map[string]any) (Ack, error) {
	res, err := conn.Call(ctx, zcode.MethodProviderUpdateAccountConfig, params)
	if err != nil {
		return Ack{}, fmt.Errorf("provider/updateAccountConfig: %w", err)
	}

	return parseAck(res)
}

// parseAck decodes the ack result {receivedRevision, providerCount,
// status} (dialect §8 bundle literal) — the map dig-out keeps the
// capture-verbatim camelCase wire keys (the tagliatelle contract).
func parseAck(res json.RawMessage) (Ack, error) {
	var obj map[string]any
	if err := json.Unmarshal(res, &obj); err != nil {
		return Ack{}, fmt.Errorf("provider/updateAccountConfig: decode ack: %w (%s)", err, res)
	}

	return Ack{
		ReceivedRevision: ackString(obj, "receivedRevision"),
		ProviderCount:    ackInt(obj, "providerCount"),
		Status:           ackString(obj, "status"),
	}, nil
}

// ackString digs one string field out of a decoded ack object.
func ackString(obj map[string]any, key string) string {
	s, _ := obj[key].(string)

	return s
}

// ackInt digs one numeric field out of a decoded ack object.
func ackInt(obj map[string]any, key string) int {
	n, _ := obj[key].(float64)

	return int(n)
}

// RefreshRegistry runs provider/refreshRegistry {reason} — step 1 of the
// PITFALL-1 verification recipe (the observed reply {"refreshed":true} is
// not asserted beyond transport success; the verdict is the read-back).
func RefreshRegistry(ctx context.Context, conn *zcode.Conn) error {
	params := map[string]any{"reason": refreshReason}

	if _, err := conn.Call(ctx, zcode.MethodProviderRefreshRegistry, params); err != nil {
		return fmt.Errorf("provider/refreshRegistry: %w", err)
	}

	return nil
}

// errAPIKeyUnconfigured is the honest unconfigured-key arm (OQ4: the error
// names the env var). It answers the coded -32603 envelope at the
// responder's deliver seam — NEVER the failure-shaped headersApplied:false
// success arm (a guaranteed turn death for account providers, PITFALL 3).
var errAPIKeyUnconfigured = errors.New("api key not configured " +
	"(set ZCODE_ACP_API_KEY or pass the api key at authenticate)")

// RuntimeHeadersHandler adapts src to the 05-01 responder seam
// (interact.WithRuntimeHeadersKeySource): with a key present the
// responder's policy row marshals the strict D-02 success shape
// {headersApplied:true, requestAuth:{apiKey, apiKeyId}} from the material
// this handler hands over (in-memory only); with no key the handler errors
// and the responder answers the coded error envelope. The api-key path's
// success arm always carries a non-empty apiKey (the mode's credential);
// apiKeyId rides whenever the source supplies it.
func RuntimeHeadersHandler(src KeySource) interact.RuntimeHeadersKeySource {
	return func(ctx context.Context, _ interact.Request) (string, string, string, error) {
		apiKey, apiKeyID, ok := src.APIKey(ctx)
		if !ok || apiKey == "" {
			return "", "", "", errAPIKeyUnconfigured
		}

		return apiKey, apiKeyID, "", nil
	}
}
