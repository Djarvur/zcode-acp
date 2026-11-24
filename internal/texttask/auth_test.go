package texttask_test

import (
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/texttask"
)

// accountProvider is the rebound account provider id the auth table names.
const accountProvider = "account:zai-individual-coding-plan"

// TestAuthMethod covers the minimal-auth decision table: account selections
// need the coding-plan chain, zai-api needs the api-key chain when a wrapper
// key is configured (the coding-plan chain when that is the selected plan),
// and any other provider needs nothing.
func TestAuthMethod(t *testing.T) {
	cases := []struct {
		name       string
		selection  acpfront.ModelSelection
		wrapperKey string
		codingPlan bool
		wantMethod string
	}{
		{
			name:       "an account selection answers the coding-plan method",
			selection:  acpfront.ModelSelection{ProviderID: accountProvider, ModelID: testModelFlash},
			wantMethod: acpfront.AuthMethodZaiCodingPlan,
		},
		{
			name:       "zai-api with a wrapper key answers the api-key method",
			selection:  acpfront.ModelSelection{ProviderID: testProviderZaiAPI, ModelID: testModelFlash},
			wrapperKey: "fixture-key",
			wantMethod: acpfront.AuthMethodZaiAPIKey,
		},
		{
			name:       "zai-api without a key on the coding plan answers the coding-plan method",
			selection:  acpfront.ModelSelection{ProviderID: testProviderZaiAPI, ModelID: testModelFlash},
			codingPlan: true,
			wantMethod: acpfront.AuthMethodZaiCodingPlan,
		},
		{
			name:       "zai-api without a key and no coding plan answers no method",
			selection:  acpfront.ModelSelection{ProviderID: testProviderZaiAPI, ModelID: testModelFlash},
			wantMethod: "",
		},
		{
			name:       "a personal provider answers no method",
			selection:  acpfront.ModelSelection{ProviderID: testProviderPersonal, ModelID: testModelPersonal},
			wrapperKey: "fixture-key",
			wantMethod: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := texttask.AuthMethod(tc.selection, tc.wrapperKey, tc.codingPlan); got != tc.wantMethod {
				t.Fatalf("AuthMethod(%+v, key=%v, plan=%v) = %q, want %q",
					tc.selection, tc.wrapperKey != "", tc.codingPlan, got, tc.wantMethod)
			}
		})
	}
}

// TestApplyCodingPlanRebind covers the mirrored rebind rule: only the
// coding-plan auth mode rewrites the builtin zai-api provider; every other
// mode (and every other provider) is the identity.
func TestApplyCodingPlanRebind(t *testing.T) {
	zai := acpfront.ModelSelection{
		ProviderID: testProviderZaiAPI, ModelID: testModelFlash, ReasoningLevel: testLevelHigh,
	}
	account := acpfront.ModelSelection{ProviderID: accountProvider, ModelID: testModelFlash}
	personal := acpfront.ModelSelection{ProviderID: testProviderPersonal, ModelID: testModelPersonal}

	cases := []struct {
		name      string
		selection acpfront.ModelSelection
		authMode  string
		want      acpfront.ModelSelection
	}{
		{"coding plan rebinds zai-api", zai, acpfront.AuthModeCodingPlan,
			acpfront.ModelSelection{
				ProviderID: accountProvider, ModelID: testModelFlash, ReasoningLevel: testLevelHigh,
			}},
		{"coding plan leaves an account selection untouched", account, acpfront.AuthModeCodingPlan, account},
		{"api-key mode leaves zai-api untouched", zai, acpfront.AuthModeAPIKey, zai},
		{"no mode leaves zai-api untouched", zai, acpfront.AuthModeNone, zai},
		{"coding plan leaves a personal provider untouched", personal, acpfront.AuthModeCodingPlan, personal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := texttask.ApplyCodingPlanRebind(tc.selection, tc.authMode); got != tc.want {
				t.Fatalf("ApplyCodingPlanRebind(%+v, %q) = %+v, want %+v", tc.selection, tc.authMode, got, tc.want)
			}
		})
	}
}
