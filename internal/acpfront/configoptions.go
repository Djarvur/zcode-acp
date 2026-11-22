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
// providerId + "/" + modelId (the repo's existing read-back serialization).
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
// verbatim: the fields the translator mirrors ride; the rest of the wire
// entry (contextWindow, properties, …) stays unparsed — the neutral list
// carries exactly what the picker renders.
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
// reply IS the confirmation source). A reply whose bytes do not decode
// answers ok=false; a decodable reply with absent/ill-typed fields answers
// the zero-value fields it could honestly capture — never a panic.
func ParseSettings(raw json.RawMessage) (Settings, bool) {
	return Settings{}, false // RED skeleton — GREEN fills the dig
}

// BuildConfigOptions translates one confirmed settings snapshot into the
// neutral configOptions vocabulary (the adapter alone assembles the SDK
// select variants — the neutral DTOs never leak SDK types, D-01). Entries
// omit honestly: the model option when the snapshot binds no current and no
// lastUsed, the thought-level option when disabled, current-empty, or
// level-source-empty.
func BuildConfigOptions(s Settings) []acpport.ConfigOptionDef {
	return nil // RED skeleton — GREEN fills the translator
}
