package interact

import (
	"context"
	"encoding/json"
)

// The production entitlement gate (AUTH-02, plan 06-05 — 06-RESEARCH.md
// Pattern 2's entitlement paragraph, the wrapper half of the account
// snapshot): a provider family is entitled for the coding-plan path when
// the desktop's OWN state says so, read READ-ONLY:
//
//   - ~/.zcode/v2/setting.json has
//     providerFamilyConnectionSelections[family].kind ==
//     "individual-coding-plan", AND
//   - modelProviderFamilyModes[family] != "apiKey" (absent entry passes —
//     the verified rule's strict-inequality semantics), AND
//   - the credential store holds oauth:<family>:access_token (the 06-01
//     reader — no parallel read path).
//
// Honest degradation (T-06-16): a missing setting.json is a fresh machine
// simply not logged in (false + a structural log); an undecodable
// setting.json answers false with a coded structural log — never a panic,
// never a fabricated true. Values never reach logs: the gate logs presence
// booleans and class codes only (T-06-15; D-12 parity — reading the desktop
// login is the credential source, NOT the api-key OQ4 rule).
//
// Family is the literal "zai" at the production call site; the gate stays
// family-generic per the reference's multi-family shape.

// The capture-verbatim literals the rule compares against (ref
// account-provider.ts oauthSelectedCodingPlanFamilies).
const (
	codingPlanSelectionKind = "individual-coding-plan"
	apiKeyFamilyMode        = "apiKey"
)

// The gate's structural-log codes (short coded tokens — never messages with
// values, never file dumps).
const (
	famselCodeSettingMissing     = "setting-missing"
	famselCodeSettingUnreadable  = "setting-unreadable"
	famselCodeSettingUndecodable = "setting-undecodable"
	famselCodeStoreUnreadable    = "store-unreadable"
)

// EntitlementGate is the production entitlement decision the account
// snapshot builder receives (hostsvc.BuildAccount's entitled func — the
// gate may only omit families, never edit entries). Constructed once via
// NewEntitlementGate; Entitled is safe for concurrent use (the 06-01
// reader's posture).
type EntitlementGate struct {
	settingPath string
	store       *CredentialStore
}

// NewEntitlementGate returns the gate over the desktop setting.json path
// and the 06-01 credential store (the ONLY store path the gate reads).
func NewEntitlementGate(settingPath string, store *CredentialStore) *EntitlementGate {
	return &EntitlementGate{settingPath: settingPath, store: store}
}

// Entitled answers whether the family carries the coding-plan entitlement
// per the desktop's own read-only state. Boolean only: every degradation
// arm answers false through the same posture, with the structural log
// carrying the class.
func (g *EntitlementGate) Entitled(ctx context.Context, family string) bool {
	return famselStubFalse(ctx, family, g)
}

// famselStubFalse is the compiling skeleton's marker (plan 06-05 RED): the
// GREEN step replaces it with the real rule.
func famselStubFalse(_ context.Context, _ string, _ *EntitlementGate) bool {
	return false
}

// settingDoc is the minimal decode of the desktop's setting.json — the
// capture-verbatim camelCase wire keys the verified rule reads. selections
// values stay raw so the kind dig can degrade per family.
type settingDoc struct {
	ProviderFamilyConnectionSelections map[string]json.RawMessage `json:"providerFamilyConnectionSelections"`
	ModelProviderFamilyModes           map[string]string          `json:"modelProviderFamilyModes"`
}
