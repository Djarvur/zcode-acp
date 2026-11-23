package interact

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
)

// The production entitlement gate (AUTH-02, plan 06-05 — 06-RESEARCH.md
// Pattern 2's entitlement paragraph, the wrapper half of the account
// snapshot; keyed per (family, mode) since quick task 261123-vyl): a
// builtin rule is entitled for the coding-plan path when the desktop's OWN
// state says the user SELECTED that rule's connection, read READ-ONLY:
//
//   - ~/.zcode/v2/setting.json has
//     providerFamilyConnectionSelections[family].kind == mode — the
//     desktop's selected connection kind IS this rule's mode
//     (individual-coding-plan, team-coding-plan and start-plan are the
//     selectable kinds; off-peak/highspeed never appear as kinds and
//     therefore never match), AND
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

// The capture-verbatim literal the rule compares against (ref
// account-provider.ts oauthSelectedCodingPlanFamilies — the apiKey mode
// veto; the selectable connection kinds are the rule's own modes).
const apiKeyFamilyMode = "apiKey"

// The gate's structural-log codes (short coded tokens — never messages with
// values, never file dumps).
const (
	famselCodeSettingMissing     = "setting-missing"
	famselCodeSettingUnreadable  = "setting-unreadable"
	famselCodeSettingUndecodable = "setting-undecodable"
)

// EntitlementGate is the production entitlement decision the account
// snapshot builder receives (hostsvc.BuildAccount's entitled func — the
// gate may only omit (family, mode) pairs, never edit entries).
// Constructed once via NewEntitlementGate; Entitled is safe for concurrent
// use (the 06-01 reader's posture).
type EntitlementGate struct {
	settingPath string
	store       *CredentialStore
}

// NewEntitlementGate returns the gate over the desktop setting.json path
// and the 06-01 credential store (the ONLY store path the gate reads).
func NewEntitlementGate(settingPath string, store *CredentialStore) *EntitlementGate {
	return &EntitlementGate{settingPath: settingPath, store: store}
}

// Entitled answers whether the desktop's own read-only state entitles this
// RULE — the (family, mode) pair, where mode is the builtin rule's
// config.access.mode. Boolean only: every degradation arm answers false
// through the same posture, with the structural log carrying the class.
// kind_ok means the selected connection kind equals the rule's mode
// exactly; no kind or mode VALUE is ever logged.
func (g *EntitlementGate) Entitled(ctx context.Context, family, mode string) bool {
	doc, ok := g.readSetting()
	if !ok {
		return false
	}

	kind, kindOK := selectionKind(doc.ProviderFamilyConnectionSelections[family])
	kindOK = kindOK && kind == mode

	// The verified rule is a strict inequality against the explicit apiKey
	// value — an ABSENT modes entry passes (JS `!== "apiKey"` semantics,
	// ref account-provider.ts oauthSelectedCodingPlanFamilies).
	familyMode, hasMode := doc.ModelProviderFamilyModes[family]
	modeOK := !hasMode || familyMode != apiKeyFamilyMode

	tokenPresent, ok := g.store.LoginToken(ctx, family)
	_ = tokenPresent // presence only — the value is never surfaced (T-06-15)

	entitled := kindOK && modeOK && ok

	slog.Info("interact: entitlement gate", "family", family,
		"kind_ok", kindOK, "mode_ok", modeOK, "token_present", ok,
		"entitled", entitled)

	return entitled
}

// readSetting reads the desktop setting.json read-only and decodes the
// minimal doc. Every degradation arm answers not-ok through the same
// posture with its structural class: a missing file is a fresh machine
// simply not logged in; an unreadable or undecodable file degrades honestly
// (false + the coded log) — never a panic, never a fabricated true, never a
// file dump in the log (path shapes and file lengths only).
func (g *EntitlementGate) readSetting() (settingDoc, bool) {
	var doc settingDoc

	raw, err := os.ReadFile(g.settingPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("interact: entitlement gate: setting.json absent (a fresh machine is simply not logged in)",
				"code", famselCodeSettingMissing, "path_len", len(g.settingPath))

			return doc, false
		}

		slog.Warn("interact: entitlement gate: setting.json unread",
			"code", famselCodeSettingUnreadable, "path_len", len(g.settingPath), "err_len", len(err.Error()))

		return doc, false
	}

	if err := json.Unmarshal(raw, &doc); err != nil {
		slog.Warn("interact: entitlement gate: setting.json undecodable",
			"code", famselCodeSettingUndecodable, "file_len", len(raw))

		return doc, false
	}

	return doc, true
}

// selectionKind digs one family's selection kind from its raw selection
// value (the capture-verbatim `kind` field). A missing family, an
// unstructured value, or an empty kind answers not-ok — an empty kind can
// never equal a rule's mode.
func selectionKind(raw json.RawMessage) (string, bool) {
	var sel struct {
		Kind string `json:"kind"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &sel) != nil || sel.Kind == "" {
		return "", false
	}

	return sel.Kind, true
}

// settingDoc is the minimal decode of the desktop's setting.json — the
// capture-verbatim camelCase wire keys the verified rule reads. selections
// values stay raw so the kind dig can degrade per family.
//
//nolint:tagliatelle // wire shape: the desktop's setting.json keys are capture-verbatim camelCase
type settingDoc struct {
	ProviderFamilyConnectionSelections map[string]json.RawMessage `json:"providerFamilyConnectionSelections"`
	ModelProviderFamilyModes           map[string]string          `json:"modelProviderFamilyModes"`
}
