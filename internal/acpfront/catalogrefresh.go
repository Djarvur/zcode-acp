package acpfront

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

const (
	catalogRefreshBudget = 15 * time.Second
	catalogWriteBudget   = 5 * time.Second
)

var errCatalogUnconfirmed = errors.New("probe session/create: no confirmed listed model selection")

// BindUpdateSink binds the connection-owned asynchronous sink before traffic.
func (f *Front) BindUpdateSink(sink acpport.UpdateSink) {
	f.catalogSink.Store(&sink)
}

// registryCatalogPatch classifies only provider-registry patches. Prompt/status
// patches and malformed payloads never generate backend work.
func registryCatalogPatch(raw json.RawMessage) bool {
	var params struct {
		Patch struct {
			Providers map[string]json.RawMessage `json:"providers"`
		} `json:"patch"`
	}

	return json.Unmarshal(raw, &params) == nil && len(params.Patch.Providers) > 0
}

func registryCatalogSignal(raw json.RawMessage) string {
	digest := sha256.Sum256(raw)

	return hex.EncodeToString(digest[:])
}

// RequestCatalogRefresh is the adapter-facing post-authenticate request.
func (f *Front) RequestCatalogRefresh(ctx context.Context) {
	f.requestCatalogRefresh(ctx, "")
}

func (f *Front) requestCatalogRefresh(ctx context.Context, signal string) {
	if ctx.Err() != nil || !f.hasActiveCatalogSession() {
		return
	}

	f.catalogRefreshMu.Lock()
	defer f.catalogRefreshMu.Unlock()

	if f.catalogRefreshRunning {
		if signal != "" && signal == f.catalogRefreshSignal {
			return
		}
		f.catalogRefreshPending = true

		return
	}

	f.catalogRefreshRunning = true
	f.catalogRefreshPending = false
	f.catalogRefreshSignal = signal

	//nolint:contextcheck,gosec // the worker intentionally owns one bounded background context
	go f.runCatalogRefresh()
}

func (f *Front) hasActiveCatalogSession() bool {
	f.settingsMu.Lock()
	ids := make([]string, 0, len(f.settings))
	for id := range f.settings {
		ids = append(ids, id)
	}
	f.settingsMu.Unlock()

	for _, id := range ids {
		if _, err := f.registry.Get(id); err == nil && !f.isClosing(id) {
			return true
		}
	}

	return false
}

// runCatalogRefresh owns one bounded context for every coalesced pass. It
// never stores a context in Front and never creates per-notification workers.
func (f *Front) runCatalogRefresh() {
	ctx, cancel := context.WithTimeout(context.Background(), catalogRefreshBudget)
	defer cancel()

	for ctx.Err() == nil {
		err := f.refreshCatalogOnce(ctx)
		if err != nil {
			slog.Warn("acpfront: live catalog refresh failed", "step", "refresh", "err", err)
		}

		f.catalogRefreshMu.Lock()
		if ctx.Err() != nil || err != nil {
			f.catalogRefreshRunning = false
			f.catalogRefreshPending = false
			f.catalogRefreshSignal = ""
			f.catalogRefreshMu.Unlock()

			return
		}

		if !f.catalogRefreshPending {
			f.catalogRefreshRunning = false
			f.catalogRefreshMu.Unlock()

			return
		}
		f.catalogRefreshPending = false
		f.catalogRefreshMu.Unlock()
	}
}

func (f *Front) refreshCatalogOnce(ctx context.Context) error {
	if err := hostsvc.RefreshRegistry(ctx, f.conn); err != nil {
		return fmt.Errorf("refresh registry: %w", err)
	}

	probe, err := f.probeCatalog(ctx)
	if err != nil {
		return err
	}

	f.publishFreshCatalog(ctx, probe)

	return nil
}

// probeCatalog reads the authoritative catalog from a fresh bare create. The
// refresh acknowledgement is deliberately discarded as catalog evidence.
func (f *Front) probeCatalog(ctx context.Context) (json.RawMessage, error) {
	dir, err := os.MkdirTemp("", "zcode-acp-catalog-*")
	if err != nil {
		return nil, fmt.Errorf("create probe directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	res, err := f.conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		wireKeyWorkspace: map[string]any{wireKeyWorkspacePath: dir, wireKeyWorkspaceKey: dir},
	})
	if err != nil {
		return nil, fmt.Errorf("probe session/create: %w", err)
	}

	sid, err := createSessionID(res, zcode.MethodSessionCreate)
	if err != nil {
		return nil, err
	}
	defer func() {
		if _, closeErr := f.conn.Call(ctx, zcode.MethodSessionClose,
			map[string]any{wireKeySessionID: sid}); closeErr != nil {
			slog.Warn("acpfront: catalog probe close failed", "session", sid, "err", closeErr)
		}
	}()

	if _, valid := snapshotForLiveCatalog(res, nil); !valid {
		return nil, errCatalogUnconfirmed
	}

	return res, nil
}

func (f *Front) publishFreshCatalog(ctx context.Context, fresh json.RawMessage) {
	f.settingsMu.Lock()
	ids := make([]string, 0, len(f.settings))
	for id := range f.settings {
		ids = append(ids, id)
	}
	f.settingsMu.Unlock()
	slices.Sort(ids)

	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		if _, err := f.registry.Get(id); err != nil || f.isClosing(id) {
			continue
		}
		previous, ok := f.latestSettings(id)
		if !ok {
			continue
		}
		merged, safe := snapshotForLiveCatalog(fresh, previous)
		if !safe {
			continue
		}
		settings, parsed := ParseSettings(merged)
		if !parsed {
			continue
		}
		options := BuildConfigOptions(settings)
		if len(options) == 0 || !listedCurrentValues(options) {
			continue
		}
		oldSettings, oldParsed := ParseSettings(previous)
		if oldParsed && reflect.DeepEqual(BuildConfigOptions(oldSettings), options) {
			continue
		}
		f.publishCatalogUpdate(ctx, id, merged, options)
	}
}

func listedCurrentValues(options []acpport.ConfigOptionDef) bool {
	for _, option := range options {
		found := false
		for _, entry := range option.Options {
			if entry.Value == option.CurrentValue {
				found = true

				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// snapshotForLiveCatalog combines fresh authoritative availability with only
// a listed prior selection or a listed backend-confirmed fallback.
func snapshotForLiveCatalog(fresh, previous json.RawMessage) (json.RawMessage, bool) {
	freshSettings, ok := ParseSettings(fresh)
	if !ok || len(freshSettings.Model.Available) == 0 {
		return nil, false
	}

	selected := confirmedListedSelection(previous, freshSettings.Model.Available)
	if selected == nil {
		selected = confirmedListedSelection(fresh, freshSettings.Model.Available)
	}
	if selected == nil {
		return nil, false
	}

	var root map[string]any
	if err := json.Unmarshal(fresh, &root); err != nil {
		return nil, false
	}
	settings, _ := root["settings"].(map[string]any)
	model, _ := settings["model"].(map[string]any)
	model["current"] = map[string]any{"providerId": selected.ProviderID, "modelId": selected.ModelID}
	delete(model, "lastUsed")
	candidate, err := json.Marshal(root)
	if err != nil {
		return nil, false
	}
	merged, ok := mergeCatalogEntries(fresh, candidate)
	if !ok {
		return nil, false
	}
	parsed, valid := ParseSettings(merged)
	if !valid || !listedCurrentValues(BuildConfigOptions(parsed)) {
		return nil, false
	}

	return merged, true
}

func confirmedListedSelection(raw json.RawMessage, available []ModelEntry) *ModelRef {
	if len(raw) == 0 {
		return nil
	}
	settings, ok := ParseSettings(raw)
	if !ok {
		return nil
	}
	for _, candidate := range []*ModelRef{settings.Model.Current, settings.Model.LastUsed} {
		if candidate == nil {
			continue
		}
		for _, entry := range available {
			if entry.Ref == *candidate {
				selected := *candidate

				return &selected
			}
		}
	}

	return nil
}

// publishCatalogUpdate is the final closing-aware gate. closingMu stays held
// through the bounded client write and settings commit, so CloseSession either
// tears down first or waits for this publication and then drops its snapshot.
func (f *Front) publishCatalogUpdate(
	ctx context.Context, sessionID string, merged json.RawMessage, options []acpport.ConfigOptionDef,
) bool {
	bound := f.catalogSink.Load()
	if bound == nil {
		return false
	}

	f.closingMu.Lock()
	defer f.closingMu.Unlock()

	if f.closing[sessionID] {
		return false
	}
	if _, err := f.registry.Get(sessionID); err != nil {
		return false
	}

	writeCtx, cancel := context.WithTimeout(ctx, catalogWriteBudget)
	defer cancel()
	if err := (*bound)(writeCtx, acpport.SessionNote{
		SessionID: sessionID, Kind: acpport.NoteConfigOptions, ConfigOptions: options,
	}); err != nil {
		slog.Warn("acpfront: catalog update write failed", "session", sessionID, "err", err)

		return false
	}

	f.recordSettings(sessionID, merged)

	return true
}
