package texttask

import "github.com/Djarvur/zcode-acp/internal/acpfront"

// The provider ids the auth decision and the coding-plan rebind speak: the
// builtin personal-layer serialization and the account provider
// internal/acpfront/front.go's applyConfiguredModel rebinds it to when the
// coding plan authenticated (the mirrored rule — this package never edits
// acpfront).
const (
	zaiAPI              = "zai-api"
	accountZaiIndividual = "account:zai-individual-coding-plan"
)

// AuthMethod decides which advertised authenticate method (if any) the
// resolved selection needs before the generateText call (stub: RED
// skeleton).
func AuthMethod(sel acpfront.ModelSelection, wrapperKey string, codingPlanSelected bool) string {
	_ = sel
	_ = wrapperKey
	_ = codingPlanSelected

	return ""
}

// ApplyCodingPlanRebind mirrors applyConfiguredModel's coding-plan rebind
// (stub: RED skeleton).
func ApplyCodingPlanRebind(sel acpfront.ModelSelection, authMode string) acpfront.ModelSelection {
	_ = authMode

	return sel
}
