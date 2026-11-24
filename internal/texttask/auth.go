package texttask

import (
	"strings"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
)

// The provider ids the auth decision and the coding-plan rebind speak: the
// builtin personal-layer serialization and the account provider
// internal/acpfront/front.go's applyConfiguredModel rebinds it to when the
// coding plan authenticated (the mirrored rule — this package never edits
// acpfront).
const (
	zaiAPI               = "zai-api"
	accountZaiIndividual = "account:zai-individual-coding-plan"

	// accountProviderPrefix marks an account-family selection: the coding
	// plan's own providers authenticate through the coding-plan chain.
	accountProviderPrefix = "account:"
)

// AuthMethod decides which advertised authenticate method (if any) the
// resolved selection needs before the generateText call:
//   - an account:* selection answers the coding-plan method;
//   - zai-api answers the api-key method when a wrapper key is configured;
//   - zai-api without a wrapper key answers the coding-plan method only when
//     that is the selected plan (the entitlement gate's answer), else no
//     method (the desktop's own provider_config credentials are the only
//     source there);
//   - any other provider (a personal provider such as OpenCodeGo) answers no
//     method: the child reads its key from provider_config.json itself.
func AuthMethod(sel acpfront.ModelSelection, wrapperKey string, codingPlanSelected bool) string {
	switch {
	case strings.HasPrefix(sel.ProviderID, accountProviderPrefix):
		return acpfront.AuthMethodZaiCodingPlan
	case sel.ProviderID != zaiAPI:
		return ""
	case wrapperKey != "":
		return acpfront.AuthMethodZaiAPIKey
	case codingPlanSelected:
		return acpfront.AuthMethodZaiCodingPlan
	default:
		return ""
	}
}

// ApplyCodingPlanRebind mirrors internal/acpfront/front.go's
// applyConfiguredModel rule (front.go:1036-1039): when the coding plan
// actually authenticated and the selection is the builtin zai-api
// serialization, the request must name the account provider — the provider
// the user's plan serves. Every other mode and provider is the identity.
func ApplyCodingPlanRebind(sel acpfront.ModelSelection, authMode string) acpfront.ModelSelection {
	if authMode == acpfront.AuthModeCodingPlan && sel.ProviderID == zaiAPI {
		sel.ProviderID = accountZaiIndividual
	}

	return sel
}
