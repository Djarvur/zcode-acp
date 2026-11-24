package acpfront

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
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

var (
	errCatalogUnconfirmed      = errors.New("probe session/create: no confirmed model catalog")
	errCatalogPublish          = errors.New("catalog update publication failed")
	errCatalogFallbackMismatch = errors.New("catalog fallback reply confirmed an unreconciled current")
)

type catalogPublishResult uint8

const (
	catalogPublishFailed catalogPublishResult = iota
	catalogPublishSkipped
	catalogPublishConfirmed
)

// BindUpdateSink binds the connection-owned asynchronous sink before traffic.
func (f *Front) BindUpdateSink(sink acpport.UpdateSink) {
	f.catalogSink.Store(&sink)
}

func (f *Front) lockCatalogSession(ctx context.Context, sessionID string) (func(), error) {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(sessionID))
	sem := f.catalogSessionLocks[int(hash.Sum32()%uint32(len(f.catalogSessionLocks)))]
	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("session stripe wait canceled: %w", ctx.Err())
	}
}

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

// requestCatalogRefresh performs only classification/hash/mutex hand-off on
// the observer read loop. Active-session and closing checks belong to workers.
func (f *Front) requestCatalogRefresh(ctx context.Context, signal string) {
	if ctx.Err() != nil {
		return
	}

	f.catalogRefreshMu.Lock()
	defer f.catalogRefreshMu.Unlock()

	if f.catalogRefreshRunning {
		if signal != "" && (signal == f.catalogRefreshSignal || signal == f.catalogPendingSignal) {
			return
		}
		f.catalogRefreshPending = true
		f.catalogPendingSignal = signal

		return
	}
	if signal != "" && signal == f.catalogRefreshSignal {
		return
	}

	f.catalogRefreshRunning = true
	f.catalogRefreshPending = false
	f.catalogPendingSignal = ""
	f.catalogRefreshSignal = signal

	//nolint:contextcheck,gosec // the worker intentionally owns one bounded background context
	go f.runCatalogRefresh()
}

func (f *Front) hasActiveCatalogSession(ctx context.Context) bool {
	f.settingsMu.Lock()
	ids := make([]string, 0, len(f.settings))
	for id := range f.settings {
		ids = append(ids, id)
	}
	f.settingsMu.Unlock()

	for _, id := range ids {
		closing, err := f.isClosingContext(ctx, id)
		if err != nil {
			return false
		}
		if _, err := f.registry.Get(id); err == nil && !closing {
			return true
		}
	}

	return false
}

// runCatalogRefresh owns one bounded context for every coalesced pass. A
// different pending signal survives a failed pass and becomes the next pass's
// identity, so it cannot repeat until the worker's bounded lifetime ends.
func (f *Front) runCatalogRefresh() {
	ctx, cancel := context.WithTimeout(context.Background(), catalogRefreshBudget)
	defer cancel()

	for ctx.Err() == nil {
		if !f.hasActiveCatalogSession(ctx) {
			f.finishCatalogRefresh(false)

			return
		}

		err := f.refreshCatalogOnce(ctx)
		if err != nil {
			slog.Warn("acpfront: live catalog refresh failed", "step", "refresh", "err", err)
		}

		f.catalogRefreshMu.Lock()
		if ctx.Err() != nil {
			f.catalogRefreshRunning = false
			f.catalogRefreshPending = false
			f.catalogRefreshSignal = ""
			f.catalogPendingSignal = ""
			f.catalogRefreshMu.Unlock()

			return
		}
		if f.catalogRefreshPending {
			f.catalogRefreshSignal = f.catalogPendingSignal
			f.catalogPendingSignal = ""
			f.catalogRefreshPending = false
			f.catalogRefreshMu.Unlock()

			continue
		}
		f.catalogRefreshRunning = false
		if err != nil {
			f.catalogRefreshSignal = ""
		}
		f.catalogRefreshMu.Unlock()

		return
	}
}

func (f *Front) finishCatalogRefresh(keepSignal bool) {
	f.catalogRefreshMu.Lock()
	defer f.catalogRefreshMu.Unlock()
	f.catalogRefreshRunning = false
	f.catalogRefreshPending = false
	if !keepSignal {
		f.catalogRefreshSignal = ""
	}
	f.catalogPendingSignal = ""
}

func (f *Front) refreshCatalogOnce(ctx context.Context) error {
	if err := hostsvc.RefreshRegistry(ctx, f.conn); err != nil {
		return fmt.Errorf("refresh registry: %w", err)
	}

	probe, err := f.probeCatalog(ctx)
	if err != nil {
		return err
	}

	return f.publishFreshCatalog(ctx, probe)
}

// probeCatalog reads authoritative availability from a fresh bare create. A
// probe with no current is valid: selection policy is per live session.
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

	settings, valid := ParseSettings(res)
	if !valid || len(settings.Model.Available) == 0 {
		return nil, errCatalogUnconfirmed
	}

	return res, nil
}

func (f *Front) publishFreshCatalog(ctx context.Context, fresh json.RawMessage) error {
	f.settingsMu.Lock()
	ids := make([]string, 0, len(f.settings))
	for id := range f.settings {
		ids = append(ids, id)
	}
	f.settingsMu.Unlock()
	slices.Sort(ids)

	var publishErr error
	for _, id := range ids {
		if ctx.Err() != nil {
			return fmt.Errorf("catalog fan-out canceled: %w", ctx.Err())
		}

		unlock, err := f.lockCatalogSession(ctx, id)
		if err != nil {
			return fmt.Errorf("catalog fan-out synchronization: %w", err)
		}
		result := f.publishFreshSession(ctx, id, fresh)
		unlock()
		if ctx.Err() != nil {
			return fmt.Errorf("catalog fan-out canceled: %w", ctx.Err())
		}
		if result == catalogPublishFailed {
			publishErr = errors.Join(publishErr, fmt.Errorf("%w: session %s", errCatalogPublish, id))
		}
	}

	return publishErr
}

// publishFreshSession recomputes from the latest snapshot while holding the
// same stripe lock as SetSessionConfigOption and /model. Closing skips do not
// abort healthy-session fan-out; only an actual write failure is retryable.
//
//nolint:cyclop,funlen,nestif,gocognit,gocyclo // each branch is a distinct confirmed-state safety gate
func (f *Front) publishFreshSession(ctx context.Context, id string, fresh json.RawMessage) catalogPublishResult {
	closing, err := f.isClosingContext(ctx, id)
	if err != nil {
		return catalogPublishFailed
	}
	if closing {
		return catalogPublishSkipped
	}
	if _, err := f.registry.Get(id); err != nil {
		return catalogPublishSkipped
	}
	previous, ok := f.latestSettings(id)
	if !ok {
		return catalogPublishSkipped
	}

	freshSettings, valid := ParseSettings(fresh)
	if !valid || len(freshSettings.Model.Available) == 0 {
		return catalogPublishSkipped
	}
	previousSettings, valid := ParseSettings(previous)
	if !valid {
		return catalogPublishSkipped
	}
	var selected *ModelRef
	if previousSettings.Model.Current == nil ||
		modelListed(*previousSettings.Model.Current, freshSettings.Model.Available) {
		selected = confirmedListedSelection(previous, freshSettings.Model.Available)
	}
	var merged json.RawMessage
	if selected != nil {
		merged, valid = snapshotForLiveCatalog(fresh, previous)
	} else {
		fallback := confirmedListedSelection(fresh, freshSettings.Model.Available)
		if fallback == nil {
			return catalogPublishSkipped
		}
		entry, found := catalogModelEntry(*fallback, freshSettings.Model.Available)
		if !found {
			return catalogPublishSkipped
		}
		confirmed, err := f.confirmCatalogFallback(ctx, id, entry)
		if errors.Is(err, errCatalogFallbackMismatch) {
			if closeErr := f.closeSessionLocked(ctx, id); closeErr != nil {
				slog.Warn("acpfront: fallback mismatch teardown failed", "session", id, "err", closeErr)
			}

			return catalogPublishSkipped
		}
		if err != nil {
			slog.Warn("acpfront: catalog fallback failed", "session", id, "err", err)

			return catalogPublishFailed
		}
		confirmedSettings, parsed := ParseSettings(confirmed)
		if !parsed || confirmedSettings.Model.Current == nil ||
			!modelListed(*confirmedSettings.Model.Current, freshSettings.Model.Available) {
			if closeErr := f.closeSessionLocked(ctx, id); closeErr != nil {
				slog.Warn("acpfront: fallback current teardown failed", "session", id, "err", closeErr)
			}

			return catalogPublishSkipped
		}
		merged, valid = snapshotFromConfirmedModel(fresh, confirmed)
	}
	if !valid {
		return catalogPublishSkipped
	}

	settings, parsed := ParseSettings(merged)
	if !parsed {
		return catalogPublishSkipped
	}
	options := BuildConfigOptions(settings)
	if len(options) == 0 || !listedCurrentValues(options) {
		return catalogPublishSkipped
	}
	if reflect.DeepEqual(BuildConfigOptions(previousSettings), options) {
		return catalogPublishSkipped
	}

	return f.publishCatalogUpdateLocked(ctx, id, merged, options)
}

// confirmCatalogFallback applies the candidate to the LIVE session. Only that
// session's setModel reply can confirm the new current and reasoning level.
func (f *Front) confirmCatalogFallback(ctx context.Context, id string, fallback ModelEntry) (json.RawMessage, error) {
	model := map[string]any{wireKeyProviderID: fallback.Ref.ProviderID, wireKeyModelID: fallback.Ref.ModelID}
	if level := fallbackReasoningLevel(fallback); level != "" {
		model["options"] = map[string]any{wireKeyReasoningLevel: level}
	}
	res, err := f.conn.Call(ctx, zcode.MethodSessionSetModel, map[string]any{
		wireKeySessionID: id, optionIDModel: model, wireKeyPersistLastUsed: false,
	})
	if err != nil {
		return nil, fmt.Errorf("live session setModel: %w", err)
	}
	settings, parsed := ParseSettings(res)
	if !parsed || settings.Model.Current == nil {
		return nil, fmt.Errorf("%w: reply carried no current", errCatalogFallbackMismatch)
	}
	if settings.ThoughtLevel.Enabled && settings.ThoughtLevel.Current != "" &&
		!thoughtLevelListed(settings, fallback.Ref) {
		res = withoutSettingsSection(res, "thoughtLevel")
	}

	return res, nil
}

func fallbackReasoningLevel(fallback ModelEntry) string {
	if fallback.DefaultLevel == "" {
		return ""
	}
	for _, level := range fallback.ReasoningLevels {
		if level.Value == fallback.DefaultLevel {
			return fallback.DefaultLevel
		}
	}

	return ""
}

func catalogModelEntry(ref ModelRef, available []ModelEntry) (ModelEntry, bool) {
	for _, entry := range available {
		if entry.Ref == ref {
			return entry, true
		}
	}

	return ModelEntry{}, false
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

// snapshotForLiveCatalog preserves a listed prior selection and its still-valid
// confirmed reasoning level over fresh availability. It never copies a probe
// current/lastUsed into a live session.
func snapshotForLiveCatalog(fresh, previous json.RawMessage) (json.RawMessage, bool) {
	freshSettings, ok := ParseSettings(fresh)
	if !ok || len(freshSettings.Model.Available) == 0 {
		return nil, false
	}
	previousSettings, valid := ParseSettings(previous)
	if !valid {
		return nil, false
	}
	if previousSettings.Model.Current != nil &&
		!modelListed(*previousSettings.Model.Current, freshSettings.Model.Available) {
		return nil, false // a disappeared current requires live setModel, even if lastUsed remains listed
	}
	selected := confirmedListedSelection(previous, freshSettings.Model.Available)
	if selected == nil {
		return nil, false
	}
	thought := settingsSection(previous, "thoughtLevel")
	merged, ok := snapshotWithSelection(fresh, *selected, thought)
	if !ok {
		return nil, false
	}
	parsed, valid := ParseSettings(merged)
	if !valid || !listedCurrentValues(BuildConfigOptions(parsed)) {
		return nil, false
	}

	return merged, true
}

// snapshotFromConfirmedModel combines fresh availability with a live setModel
// reply. Changed-model reasoning comes only from that reply; absent, disabled,
// or invalid enabled reasoning omits the option rather than borrowing probe state.
func snapshotFromConfirmedModel(fresh, confirmed json.RawMessage) (json.RawMessage, bool) {
	freshSettings, ok := ParseSettings(fresh)
	if !ok || len(freshSettings.Model.Available) == 0 {
		return nil, false
	}
	settings, parsed := ParseSettings(confirmed)
	if !parsed || settings.Model.Current == nil ||
		!modelListed(*settings.Model.Current, freshSettings.Model.Available) {
		return nil, false
	}
	thought := settingsSection(confirmed, "thoughtLevel")
	merged, ok := snapshotWithSelection(fresh, *settings.Model.Current, thought)
	if !ok {
		return nil, false
	}
	mergedSettings, valid := ParseSettings(merged)
	if !valid || !listedCurrentValues(BuildConfigOptions(mergedSettings)) {
		return nil, false
	}

	return merged, true
}

func snapshotWithSelection(
	fresh json.RawMessage, selected ModelRef, thought json.RawMessage,
) (json.RawMessage, bool) {
	var root map[string]any
	if err := json.Unmarshal(fresh, &root); err != nil {
		return nil, false
	}
	settings, _ := root["settings"].(map[string]any)
	model, _ := settings["model"].(map[string]any)
	model["current"] = map[string]any{"providerId": selected.ProviderID, "modelId": selected.ModelID}
	delete(model, "lastUsed")
	if len(thought) > 0 && string(thought) != jsonNullValue {
		settings["thoughtLevel"] = thought
	} else {
		delete(settings, "thoughtLevel")
	}
	candidate, err := json.Marshal(root)
	if err != nil {
		return nil, false
	}
	parsed, valid := ParseSettings(candidate)
	if !valid {
		return nil, false
	}
	if parsed.ThoughtLevel.Enabled && parsed.ThoughtLevel.Current != "" &&
		!thoughtLevelListed(parsed, selected) {
		delete(settings, "thoughtLevel")
	}
	out, err := json.Marshal(root)
	if err != nil {
		return nil, false
	}

	return out, true
}

func withoutSettingsSection(raw json.RawMessage, key string) json.RawMessage {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return raw
	}
	settings, _ := root["settings"].(map[string]any)
	if settings == nil {
		return raw
	}
	delete(settings, key)
	out, err := json.Marshal(root)
	if err != nil {
		return raw
	}

	return out
}

func settingsSection(raw json.RawMessage, key string) json.RawMessage {
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return nil
	}
	var settings map[string]json.RawMessage
	if json.Unmarshal(root["settings"], &settings) != nil {
		return nil
	}

	return settings[key]
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
		if candidate != nil && modelListed(*candidate, available) {
			selected := *candidate

			return &selected
		}
	}

	return nil
}

func modelListed(ref ModelRef, available []ModelEntry) bool {
	for _, entry := range available {
		if entry.Ref == ref {
			return true
		}
	}

	return false
}

func thoughtLevelListed(settings Settings, model ModelRef) bool {
	if !settings.ThoughtLevel.Enabled || settings.ThoughtLevel.Current == "" {
		return true
	}
	for _, entry := range settings.Model.Available {
		if entry.Ref == model && len(entry.ReasoningLevels) > 0 {
			for _, level := range entry.ReasoningLevels {
				if level.Value == settings.ThoughtLevel.Current {
					return true
				}
			}

			return false
		}
	}
	for _, level := range settings.ThoughtLevel.Available {
		if level.Value == settings.ThoughtLevel.Current {
			return true
		}
	}

	return false
}

// publishCatalogUpdate is the final closing-aware gate used by direct tests.
func (f *Front) publishCatalogUpdate(
	ctx context.Context, sessionID string, merged json.RawMessage, options []acpport.ConfigOptionDef,
) bool {
	unlock, err := f.lockCatalogSession(ctx, sessionID)
	if err != nil {
		return false
	}
	defer unlock()

	return f.publishCatalogUpdateLocked(ctx, sessionID, merged, options) == catalogPublishConfirmed
}

func (f *Front) publishCatalogUpdateLocked(
	ctx context.Context, sessionID string, merged json.RawMessage, options []acpport.ConfigOptionDef,
) catalogPublishResult {
	bound := f.catalogSink.Load()
	if bound == nil {
		return catalogPublishFailed
	}

	closing, err := f.isClosingContext(ctx, sessionID)
	if err != nil {
		return catalogPublishFailed
	}
	if closing {
		return catalogPublishSkipped
	}
	if _, err := f.registry.Get(sessionID); err != nil {
		return catalogPublishSkipped
	}

	writeCtx, cancel := context.WithTimeout(ctx, catalogWriteBudget)
	defer cancel()
	if err := (*bound)(writeCtx, acpport.SessionNote{
		SessionID: sessionID, Kind: acpport.NoteConfigOptions, ConfigOptions: options,
	}); err != nil {
		slog.Warn("acpfront: catalog update write failed", "session", sessionID, "err", err)

		return catalogPublishFailed
	}

	f.recordSettings(sessionID, merged)

	return catalogPublishConfirmed
}
