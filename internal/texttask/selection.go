package texttask

import (
	"errors"

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

// ResolveSelection returns the configured selection when present; otherwise
// it reads config.defaultModelSelection from the desktop provider_config
// file (stub: RED skeleton).
func ResolveSelection(configured *acpfront.ModelSelection, defaultSelectionPath string) (acpfront.ModelSelection, error) {
	_ = configured
	_ = defaultSelectionPath

	return acpfront.ModelSelection{}, nil
}
