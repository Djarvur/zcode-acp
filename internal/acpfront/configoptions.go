package acpfront

import (
	"encoding/json"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// configoptions.go is the MODEL-01 translator (plan 06-04): the backend
// reply's settings snapshot -> the neutral configOptions list. Its ONLY
// inputs are the snapshot fields (D-08 — the verbatim mirror: no
// wrapper-side filtering, renaming, or enrichment; a disabledReason-
// carrying entry passes through as reported) and its currentValue comes
// ONLY from server-confirmed state (D-10 — settings.model.current/lastUsed
// and settings.thoughtLevel.current under enabled; when a pair reports no
// current the whole entry is OMITTED — locked OQ2: a smaller honest list
// beats a fabricated selection).
//
// The dig discipline is digCreateAvailable's (internal/hostsvc/readback.go):
// map-based, type-assert with comma-ok, capture-verbatim camelCase wire
// keys, undecodable = absent — never a verdict, never a panic.

// ModelRef is one backend model reference (settings.model.current/lastUsed
// and available[].ref): the providerId/modelId pair the wire serializes as
// providerId + "/" + modelId (the repo's existing read-back serialization,
// readback.go/main.go).
type ModelRef struct {
	ProviderID string
	ModelID    string
}

// ReasoningLevel is one per-model reasoning level (settings.model
// .available[].reasoning.levels[]) — the D-09 source of the thought-level
// picker's entries.
type ReasoningLevel struct {
	Value       string
	Label       string
	Description string
}

// ModelEntry is one available model (settings.model.available[]), captured
// verbatim: exactly the fields the translator mirrors ride; the rest of the
// wire entry (contextWindow, properties, …) stays unparsed — the neutral
// list carries what the picker renders, nothing more.
type ModelEntry struct {
	Ref             ModelRef
	Label           string
	Description     string
	ReasoningLevels []ReasoningLevel
	DefaultLevel    string
}

// LevelEntry is one thought level (settings.thoughtLevel.available[]).
type LevelEntry struct {
	Value       string
	Label       string
	Description string
}

// ModelSettings is settings.model (RESEARCH Pattern 6 schema strict).
type ModelSettings struct {
	Current   *ModelRef
	LastUsed  *ModelRef
	Available []ModelEntry
}

// ThoughtLevelSettings is settings.thoughtLevel.
type ThoughtLevelSettings struct {
	Enabled      bool
	Current      string
	DefaultLevel string
	Available    []LevelEntry
}

// Settings is one backend settings snapshot (Pattern 6 schema): the
// translator's entire input universe. Zero-value fields are ABSENT fields —
// the omission arms are the D-10 honesty mechanism.
type Settings struct {
	Model        ModelSettings
	ThoughtLevel ThoughtLevelSettings
}

// ParseSettings digs the settings snapshot out of a backend reply result
// (create/setModel/setThoughtLevel — Pattern 7's reply-snapshot fact: the
// reply IS the confirmation source, Pitfall 6). A reply whose bytes do not
// decode answers ok=false; a decodable reply answers every field it can
// honestly capture — ill-typed or absent fields degrade to their zero
// values, never a panic.
func ParseSettings(raw json.RawMessage) (Settings, bool) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return Settings{}, false
	}

	settings, _ := obj["settings"].(map[string]any)

	s := Settings{
		Model:        digModelSettings(settings["model"]),
		ThoughtLevel: digThoughtLevelSettings(settings["thoughtLevel"]),
	}

	return s, true
}

// digModelSettings captures settings.model: the confirmed refs, the
// available list in wire order, each entry's per-model reasoning block.
func digModelSettings(raw any) ModelSettings {
	m, _ := raw.(map[string]any)
	if m == nil {
		return ModelSettings{}
	}

	var out ModelSettings

	out.Current, _ = digModelRef(m["current"])
	out.LastUsed, _ = digModelRef(m["lastUsed"])

	list, _ := m["available"].([]any)

	for _, entry := range list {
		e, _ := entry.(map[string]any)
		if e == nil {
			continue
		}

		ref, ok := digModelRef(e["ref"])
		if !ok {
			continue // a ref-less entry has no wire identity (the readback skip discipline)
		}

		me := ModelEntry{
			Ref:         *ref,
			Label:       digString(e["label"]),
			Description: digString(e["description"]),
		}

		if reasoning, _ := e["reasoning"].(map[string]any); reasoning != nil {
			me.ReasoningLevels = digLevels(reasoning["levels"])
			me.DefaultLevel = digString(reasoning["defaultLevel"])
		}

		out.Available = append(out.Available, me)
	}

	return out
}

// digModelRef captures one providerId/modelId pair; a missing or ref-less
// object is an absent reference (an empty ref cannot serialize — the
// digCreateAvailable identity discipline).
func digModelRef(raw any) (*ModelRef, bool) {
	m, _ := raw.(map[string]any)
	if m == nil {
		return nil, false
	}

	pid, _ := m["providerId"].(string)
	mid, _ := m["modelId"].(string)

	if pid == "" && mid == "" {
		return nil, false
	}

	return &ModelRef{ProviderID: pid, ModelID: mid}, true
}

// digLevels captures one reasoning.levels list in wire order; entries
// without a value are skipped (value is the level's wire identity — a
// nameless level cannot round-trip a selection).
func digLevels(raw any) []ReasoningLevel {
	list, _ := raw.([]any)
	if len(list) == 0 {
		return nil
	}

	out := make([]ReasoningLevel, 0, len(list))

	for _, entry := range list {
		e, _ := entry.(map[string]any)
		if e == nil {
			continue
		}

		v, _ := e["value"].(string)
		if v == "" {
			continue
		}

		out = append(out, ReasoningLevel{
			Value:       v,
			Label:       digString(e["label"]),
			Description: digString(e["description"]),
		})
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// digThoughtLevelSettings captures settings.thoughtLevel; an ill-typed or
// absent block is the zero value (disabled, current empty — the omission
// arms).
func digThoughtLevelSettings(raw any) ThoughtLevelSettings {
	m, _ := raw.(map[string]any)
	if m == nil {
		return ThoughtLevelSettings{}
	}

	out := ThoughtLevelSettings{
		Enabled:      digBool(m["enabled"]),
		Current:      digString(m["current"]),
		DefaultLevel: digString(m["defaultLevel"]),
	}

	list, _ := m["available"].([]any)

	for _, entry := range list {
		e, _ := entry.(map[string]any)
		if e == nil {
			continue
		}

		v, _ := e["value"].(string)
		if v == "" {
			continue
		}

		out.Available = append(out.Available, LevelEntry{
			Value:       v,
			Label:       digString(e["label"]),
			Description: digString(e["description"]),
		})
	}

	return out
}

// digString is the comma-ok string assertion (the dig discipline's scalar).
func digString(raw any) string {
	s, _ := raw.(string)

	return s
}

// digBool is the comma-ok bool assertion.
func digBool(raw any) bool {
	b, _ := raw.(bool)

	return b
}

// BuildConfigOptions translates one confirmed settings snapshot into the
// neutral configOptions vocabulary (the adapter alone assembles the SDK
// select variants — the neutral DTOs never leak SDK types across the seam,
// D-01). Options render in the translation table's order (model, then
// thought level) and omit honestly: the model option when the snapshot
// binds no current and no lastUsed, the thought-level option when
// disabled, current-empty, or level-source-empty.
func BuildConfigOptions(s Settings) []acpport.ConfigOptionDef {
	var out []acpport.ConfigOptionDef

	if model := buildModelOption(s); model != nil {
		out = append(out, *model)
	}

	if thought := buildThoughtLevelOption(s); thought != nil {
		out = append(out, *thought)
	}

	return out
}

// buildModelOption renders the model picker from the snapshot's available
// list (D-08 verbatim: Value = providerId + "/" + modelId, Name/Description
// exactly as reported — the disabledReason entry stays). currentValue is
// the confirmed binding: settings.model.current, falling back to lastUsed;
// when neither exists the whole option is OMITTED (D-10 + locked OQ2 —
// never a fabricated selection).
func buildModelOption(s Settings) *acpport.ConfigOptionDef {
	bound := s.Model.Current
	if bound == nil {
		bound = s.Model.LastUsed
	}

	if bound == nil {
		return nil
	}

	opts := make([]acpport.SelectOptionDef, 0, len(s.Model.Available))

	for _, entry := range s.Model.Available {
		opts = append(opts, acpport.SelectOptionDef{
			Value:       entry.Ref.ProviderID + "/" + entry.Ref.ModelID,
			Name:        entry.Label,
			Description: entry.Description,
		})
	}

	if len(opts) == 0 {
		return nil // an empty select renders nothing honestly
	}

	return &acpport.ConfigOptionDef{
		ID:           "model",
		Name:         "Model",
		Category:     acpport.ConfigCategoryModel,
		CurrentValue: bound.ProviderID + "/" + bound.ModelID,
		Options:      opts,
	}
}

// buildThoughtLevelOption renders the thought-level picker. Its entries are
// the BOUND model's reasoning.levels (D-09 — per-model metadata as
// reported), falling back to settings.thoughtLevel.available only when the
// bound model carries none; currentValue is settings.thoughtLevel.current
// ONLY under enabled (D-10) — a disabled or current-less block omits the
// whole option.
func buildThoughtLevelOption(s Settings) *acpport.ConfigOptionDef {
	if !s.ThoughtLevel.Enabled || s.ThoughtLevel.Current == "" {
		return nil
	}

	levels := boundModelLevels(s)
	if len(levels) == 0 {
		return nil // no level source — nothing to render honestly
	}

	return &acpport.ConfigOptionDef{
		ID:           "thought_level",
		Name:         "Thought level",
		Category:     acpport.ConfigCategoryThoughtLevel,
		CurrentValue: s.ThoughtLevel.Current,
		Options:      levels,
	}
}

// boundModelLevels resolves the level entries from the snapshot's bound
// model (current, falling back to lastUsed): the model's own
// reasoning.levels verbatim; the thoughtLevel.available fallback fires only
// when the bound model carries no levels (or is not listed).
func boundModelLevels(s Settings) []acpport.SelectOptionDef {
	bound := s.Model.Current
	if bound == nil {
		bound = s.Model.LastUsed
	}

	if bound != nil {
		for _, entry := range s.Model.Available {
			if entry.Ref == *bound && len(entry.ReasoningLevels) > 0 {
				return levelsToOptions(entry.ReasoningLevels)
			}
		}
	}

	return levelsToOptionsFromAvailable(s.ThoughtLevel.Available)
}

// levelsToOptions mirrors per-model reasoning levels into select entries.
func levelsToOptions(levels []ReasoningLevel) []acpport.SelectOptionDef {
	if len(levels) == 0 {
		return nil
	}

	opts := make([]acpport.SelectOptionDef, 0, len(levels))
	for _, l := range levels {
		opts = append(opts, acpport.SelectOptionDef{
			Value:       l.Value,
			Name:        l.Label,
			Description: l.Description,
		})
	}

	return opts
}

// levelsToOptionsFromAvailable mirrors the thoughtLevel.available fallback
// list into select entries.
func levelsToOptionsFromAvailable(levels []LevelEntry) []acpport.SelectOptionDef {
	if len(levels) == 0 {
		return nil
	}

	opts := make([]acpport.SelectOptionDef, 0, len(levels))
	for _, l := range levels {
		opts = append(opts, acpport.SelectOptionDef{
			Value:       l.Value,
			Name:        l.Label,
			Description: l.Description,
		})
	}

	return opts
}
