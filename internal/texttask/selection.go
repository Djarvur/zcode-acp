package texttask

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
)

// Selection sentinels: the messages name the path and the actionable fix
// (select a model in the client or pass --model), never file content.
var (
	// ErrNoSelection reports a missing or undecodable default-selection
	// source.
	ErrNoSelection = errors.New("text task: no model selection resolved")
	// ErrInvalidDefaultSelection reports a decoded default selection with an
	// empty provider or model id.
	ErrInvalidDefaultSelection = errors.New("text task: invalid default model selection")
)

// selectionHint is the actionable fix both selection errors carry (the
// realdesktop value is the documented "Config default" source).
const selectionHint = "select a model in the client or pass --model"

// providerConfigFile is the minimal provider_config.json decode surface:
// ONLY config.defaultModelSelection is read — provider credentials and every
// other field never enter this struct (T-261124-03's reader-side twin).
type providerConfigFile struct {
	Config struct {
		Default providerConfigSelection `json:"defaultModelSelection"` //nolint:tagliatelle // wire key
	} `json:"config"`
}

// providerConfigSelection is the desktop default-selection object.
type providerConfigSelection struct {
	ProviderID string `json:"providerId"` //nolint:tagliatelle // capture-verbatim
	ModelID    string `json:"modelId"`    //nolint:tagliatelle // capture-verbatim
	Options    struct {
		ReasoningLevel string `json:"reasoningLevel"` //nolint:tagliatelle // capture-verbatim
	} `json:"options"`
}

// ResolveSelection returns the configured selection (the --model flag or its
// env alias, already parsed by the caller) when present; otherwise it reads
// config.defaultModelSelection from the desktop provider_config file. A
// missing/invalid default is a sentinel with an actionable message — the
// wrapper never silently picks a model.
func ResolveSelection(
	configured *acpfront.ModelSelection, defaultSelectionPath string,
) (acpfront.ModelSelection, error) {
	if configured != nil {
		return *configured, nil
	}

	data, err := os.ReadFile(defaultSelectionPath)
	if err != nil {
		return acpfront.ModelSelection{}, fmt.Errorf(
			"%w: read the desktop default model selection %s: %w (%s)",
			ErrNoSelection, defaultSelectionPath, err, selectionHint)
	}

	var parsed providerConfigFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		return acpfront.ModelSelection{}, fmt.Errorf(
			"%w: decode %s: %w (%s)", ErrNoSelection, defaultSelectionPath, err, selectionHint)
	}

	def := parsed.Config.Default

	sel := acpfront.ModelSelection{
		ProviderID:     def.ProviderID,
		ModelID:        def.ModelID,
		ReasoningLevel: def.Options.ReasoningLevel,
	}

	if strings.TrimSpace(sel.ProviderID) == "" || strings.TrimSpace(sel.ModelID) == "" {
		return acpfront.ModelSelection{}, fmt.Errorf(
			"%w: %s carries an empty provider or model id (%s)",
			ErrInvalidDefaultSelection, defaultSelectionPath, selectionHint)
	}

	return sel, nil
}
