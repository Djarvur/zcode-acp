package acpfront

import (
	"context"
	"encoding/json"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

func (f *Front) BindUpdateSink(acpport.UpdateSink) {}

func registryCatalogPatch(raw json.RawMessage) bool {
	return false
}

func snapshotForLiveCatalog(fresh, previous json.RawMessage) (json.RawMessage, bool) {
	return nil, false
}

func (f *Front) publishCatalogUpdate(
	ctx context.Context, sessionID string, merged json.RawMessage, options []acpport.ConfigOptionDef,
) bool {
	return false
}
