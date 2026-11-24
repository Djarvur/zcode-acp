package acpfront_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// Config-options translator vectors (plan 06-04 Task 1, MODEL-01): the
// settings snapshot -> configOptions translator is locked against fixture
// reply snapshots shaped like the live create/setModel reply (RESEARCH
// Pattern 6 schema strict). The discipline is the verbatim-mirror test
// shape (06-03 precedent): every emitted Value/Name/Description traces
// element-for-element to the fixture snapshot (D-08 — no filtering,
// renaming, enrichment), the thought-level entries come from the bound
// model's reasoning.levels with the thoughtLevel.available fallback (D-09),
// and currentValue appears ONLY from server-confirmed fields (D-10) — the
// omission arms are the honesty mechanism (locked OQ2: a smaller honest
// list beats a fabricated selection).
//
// Each fixture is a REPLY RESULT (the `settings` key rides inside it — the
// create/setModel reply shape), because ParseSettings consumes reply
// snapshots exactly as the front captured them.

// translatorFixture is the primary verbatim-mirror snapshot: model.current
// bound to one available entry, three available entries (per-model
// reasoning levels / plain / disabledReason-carrying), and a thoughtLevel
// block. The provider ids mix zai-api and an account prefix (the D-11
// both-paths posture starts here — the translator renders whatever the
// backend reports, api-key or coding-plan).
const translatorFixture = `{
  "session": {"sessionId": "sess_fix1"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"},
      "available": [
        {
          "ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"},
          "label": "GLM-5.3-Flash",
          "description": "The flagship reasoning model",
          "contextWindow": 200000,
          "reasoning": {
            "levels": [
              {"value": "high", "label": "High", "description": "deep reasoning"},
              {"value": "low", "label": "Low"}
            ],
            "defaultLevel": "high"
          }
        },
        {
          "ref": {"providerId": "zai-api", "modelId": "GLM-5.3"},
          "label": "GLM-5.3"
        },
        {
          "ref": {"providerId": "zai-account", "modelId": "GLM-5-turbo"},
          "label": "GLM-5 Turbo",
          "description": "Legacy plan model",
          "disabledReason": "deprecated on this account"
        }
      ]
    },
    "thoughtLevel": {
      "enabled": true,
      "current": "high",
      "defaultLevel": "high",
      "available": [
        {"value": "off", "label": "Off"},
        {"value": "high", "label": "High", "description": "deep reasoning"}
      ]
    }
  }
}`

// TestConfigOptionsTranslator is the plan 06-04 Task 1 RED target: the
// settings -> configOptions translator over the fixture snapshots.
//
//nolint:funlen,gocognit,gocyclo,cyclop,maintidx // one lock per subtest; splitting would scatter contract commentary
func TestConfigOptionsTranslator(t *testing.T) {
	t.Run("verbatim mirror of the settings snapshot (D-08)", func(t *testing.T) {
		s, ok := acpfront.ParseSettings(json.RawMessage(translatorFixture))
		if !ok {
			t.Fatal("ParseSettings: the decodable reply snapshot must parse ok")
		}

		got := acpfront.BuildConfigOptions(s)

		if len(got) != 2 {
			t.Fatalf("BuildConfigOptions = %d options, want 2 (model + thought_level): %+v", len(got), got)
		}

		model := got[0]
		if model.ID != optionIDModel || model.Category != acpport.ConfigCategoryModel {
			t.Fatalf("options[0] = {ID %q Category %q}, want {model, %s}",
				model.ID, model.Category, acpport.ConfigCategoryModel)
		}

		if model.CurrentValue != corpusModelFlashID {
			t.Fatalf("model CurrentValue = %q, want the server-confirmed settings.model.current "+
				"serialization zai-api/GLM-5.3-Flash (D-10)", model.CurrentValue)
		}

		// D-08 verbatim mirror: one entry per available model, wire order,
		// Value = providerId + "/" + modelId, Name/Description verbatim —
		// the disabledReason-carrying entry PRESENT as reported (never
		// filtered); contextWindow/reasoning/defaultLevel never enriched
		// into the entries.
		want := []acpport.SelectOptionDef{
			{Value: corpusModelFlashID, Name: "GLM-5.3-Flash", Description: "The flagship reasoning model"},
			{Value: corpusModelPlainID, Name: "GLM-5.3"},
			{Value: corpusAccountModelID, Name: "GLM-5 Turbo", Description: "Legacy plan model"},
		}
		assertSelectOptions(t, model.Options, want)

		thought := got[1]
		if thought.ID != optionIDThoughtLevel || thought.Category != acpport.ConfigCategoryThoughtLevel {
			t.Fatalf("options[1] = {ID %q Category %q}, want {thought_level, %s}",
				thought.ID, thought.Category, acpport.ConfigCategoryThoughtLevel)
		}

		if thought.CurrentValue != corpusReasoningLevel {
			t.Fatalf("thought_level CurrentValue = %q, want the confirmed settings.thoughtLevel.current %q (D-10)",
				thought.CurrentValue, corpusReasoningLevel)
		}

		// D-09: the BOUND model's reasoning.levels drive the entries — the
		// thoughtLevel.available fallback list (with its off level) is NOT
		// used while the bound model carries levels.
		wantLevels := []acpport.SelectOptionDef{
			{Value: corpusReasoningLevel, Name: reasoningLabelHigh, Description: "deep reasoning"},
			{Value: reasoningLevelLow, Name: reasoningLabelLow},
		}
		assertSelectOptions(t, thought.Options, wantLevels)
	})

	t.Run("thoughtLevel.available fallback when the bound model carries no reasoning block (D-09)", func(t *testing.T) {
		s, ok := acpfront.ParseSettings(json.RawMessage(`{
		  "settings": {
		    "model": {
		      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
		      "available": [
		        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
		      ]
		    },
		    "thoughtLevel": {
		      "enabled": true,
		      "current": "low",
		      "defaultLevel": "low",
		      "available": [
		        {"value": "off", "label": "Off", "description": "no reasoning"},
		        {"value": "low", "label": "Low"}
		      ]
		    }
		  }
		}`))
		if !ok {
			t.Fatal("ParseSettings: the decodable reply snapshot must parse ok")
		}

		got := acpfront.BuildConfigOptions(s)

		if len(got) != 2 {
			t.Fatalf("BuildConfigOptions = %d options, want 2: %+v", len(got), got)
		}

		thought := got[1]
		if thought.ID != optionIDThoughtLevel {
			t.Fatalf("options[1].ID = %q, want thought_level", thought.ID)
		}

		// The fallback fires ONLY because the bound model carries no
		// reasoning block: entries now come from settings.thoughtLevel
		// .available verbatim.
		wantLevels := []acpport.SelectOptionDef{
			{Value: "off", Name: "Off", Description: "no reasoning"},
			{Value: reasoningLevelLow, Name: "Low"},
		}
		assertSelectOptions(t, thought.Options, wantLevels)

		if thought.CurrentValue != reasoningLevelLow {
			t.Fatalf("thought_level CurrentValue = %q, want low (D-10)", thought.CurrentValue)
		}
	})

	t.Run("no current and no lastUsed omits the model entry entirely (D-10 + locked OQ2)", func(t *testing.T) {
		s, ok := acpfront.ParseSettings(json.RawMessage(`{
		  "settings": {
		    "model": {
		      "available": [
		        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
		      ]
		    },
		    "thoughtLevel": {
		      "enabled": true,
		      "current": "high",
		      "available": [{"value": "high", "label": "High"}]
		    }
		  }
		}`))
		if !ok {
			t.Fatal("ParseSettings: the decodable reply snapshot must parse ok")
		}

		got := acpfront.BuildConfigOptions(s)

		if len(got) != 1 {
			t.Fatalf("BuildConfigOptions = %d options, want 1 (the model entry OMITTED — "+
				"a smaller honest list beats a fabricated currentValue): %+v", len(got), got)
		}

		if got[0].ID != optionIDThoughtLevel {
			t.Fatalf("the surviving option = %q, want thought_level", got[0].ID)
		}
	})

	t.Run("lastUsed binds the model currentValue when current is absent (D-10 fallback)", func(t *testing.T) {
		s, ok := acpfront.ParseSettings(json.RawMessage(`{
		  "settings": {
		    "model": {
		      "lastUsed": {"providerId": "zai-api", "modelId": "GLM-5.3"},
		      "available": [
		        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
		      ]
		    }
		  }
		}`))
		if !ok {
			t.Fatal("ParseSettings: the decodable reply snapshot must parse ok")
		}

		got := acpfront.BuildConfigOptions(s)

		if len(got) != 1 || got[0].ID != optionIDModel {
			t.Fatalf("BuildConfigOptions = %+v, want the model entry only (no thoughtLevel block in the snapshot)", got)
		}

		if got[0].CurrentValue != corpusModelPlainID {
			t.Fatalf("model CurrentValue = %q, want the lastUsed fallback serialization zai-api/GLM-5.3 (D-10)",
				got[0].CurrentValue)
		}
	})

	t.Run("thoughtLevel disabled omits the thought-level entry (D-10)", func(t *testing.T) {
		s, ok := acpfront.ParseSettings(json.RawMessage(`{
		  "settings": {
		    "model": {
		      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
		      "available": [
		        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
		      ]
		    },
		    "thoughtLevel": {
		      "enabled": false,
		      "current": "high",
		      "available": [{"value": "high", "label": "High"}]
		    }
		  }
		}`))
		if !ok {
			t.Fatal("ParseSettings: the decodable reply snapshot must parse ok")
		}

		got := acpfront.BuildConfigOptions(s)

		if len(got) != 1 || got[0].ID != optionIDModel {
			t.Fatalf("BuildConfigOptions = %+v, want the model entry only "+
				"(enabled:false omits the thought-level entry)", got)
		}
	})

	t.Run("thoughtLevel with an empty current omits the thought-level entry (D-10)", func(t *testing.T) {
		s, ok := acpfront.ParseSettings(json.RawMessage(`{
		  "settings": {
		    "model": {
		      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
		      "available": [
		        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
		      ]
		    },
		    "thoughtLevel": {
		      "enabled": true,
		      "current": "",
		      "available": [{"value": "high", "label": "High"}]
		    }
		  }
		}`))
		if !ok {
			t.Fatal("ParseSettings: the decodable reply snapshot must parse ok")
		}

		got := acpfront.BuildConfigOptions(s)

		if len(got) != 1 || got[0].ID != optionIDModel {
			t.Fatalf("BuildConfigOptions = %+v, want the model entry only "+
				"(empty current omits the thought-level entry)", got)
		}
	})

	for _, tc := range []struct {
		name    string
		enabled bool
		current string
	}{
		{name: "thought-disabled alias snapshot still omits thought entirely", current: corpusReasoningLevel},
		{name: "current-empty alias snapshot still omits thought entirely", enabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bound := acpfront.ModelRef{
				ProviderID: reasoningSnapshotTargetProvider, ModelID: reasoningSnapshotTargetModel,
			}
			entry := cfgReasoningEntry(bound, reasoningLevelOff, corpusReasoningLevel)
			entry.DefaultLevel = corpusReasoningLevel
			raw := cfgReasoningCreateResult(t, &bound, nil, tc.current, tc.enabled, []acpfront.ModelEntry{entry},
				[]acpfront.LevelEntry{
					{Value: orcaThoughtEnabled, Label: orcaThoughtEnabled},
					{Value: orcaThoughtDisabled, Label: orcaThoughtDisabled},
				})

			s, ok := acpfront.ParseSettings(json.RawMessage(raw))
			if !ok {
				t.Fatal("ParseSettings: the alias-era snapshot must parse ok")
			}

			got := acpfront.BuildConfigOptions(s)
			if len(got) != 1 || got[0].ID != optionIDModel {
				t.Fatalf("BuildConfigOptions = %+v, want the model entry only "+
					"(no synthesized thought alias option)", got)
			}
		})
	}

	t.Run("a snapshot missing the thoughtLevel block omits the thought-level entry", func(t *testing.T) {
		s, ok := acpfront.ParseSettings(json.RawMessage(`{
		  "settings": {
		    "model": {
		      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
		      "available": [
		        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
		      ]
		    }
		  }
		}`))
		if !ok {
			t.Fatal("ParseSettings: the decodable reply snapshot must parse ok")
		}

		got := acpfront.BuildConfigOptions(s)

		if len(got) != 1 || got[0].ID != optionIDModel {
			t.Fatalf("BuildConfigOptions = %+v, want the model entry only "+
				"(no thoughtLevel block, nothing synthesized)", got)
		}
	})

	t.Run("malformed snapshots degrade to absent fields, never a panic", func(t *testing.T) {
		for name, raw := range map[string]string{
			"empty":       ``,
			"null":        jsonNullLiteral,
			"garbage":     `{not json at all`,
			"wrong-shape": `[1,2,3]`,
			"wrong-typed": `{"settings":{"model":{"current":42,"available":"nope","lastUsed":true},` +
				`"thoughtLevel":{"enabled":"yes","current":7,"available":9}}}`,
			"missing-refs": `{"settings":{"model":{"current":{},"available":[{"ref":{}}]}}}`,
			"partial-loss": `{"settings":{"model":{"current":{"providerId":"zai-api"},` +
				`"available":[{"ref":{"providerId":"zai-api"}}]}}}`,
			"level-garbage": `{"settings":{"model":{"current":{"providerId":"zai-api","modelId":"GLM-5.3"},` +
				`"available":[{"ref":{"providerId":"zai-api","modelId":"GLM-5.3"},"label":"GLM-5.3",` +
				`"reasoning":{"levels":"nope"}}]}}}`,
		} {
			t.Run(name, func(t *testing.T) {
				// A panic anywhere in the dig or the translate fails the
				// subtest — malformed wire bytes are absent fields, never a
				// crash (the digCreateAvailable capture-verbatim posture).
				s, _ := acpfront.ParseSettings(json.RawMessage(raw))
				_ = acpfront.BuildConfigOptions(s) // the omission arms render honestly
			})
		}
	})
}

func TestParseSettingsRejectsPartialModelRefs(t *testing.T) {
	for name, raw := range map[string]string{
		"provider-only-current": `{"settings":{"model":{"current":{"providerId":"zai-api"},` +
			`"available":[{"ref":{"providerId":"zai-api","modelId":"GLM-5.3"},"label":"GLM-5.3"}]}}}`,
		"model-only-current": `{"settings":{"model":{"current":{"modelId":"GLM-5.3"},` +
			`"available":[{"ref":{"providerId":"zai-api","modelId":"GLM-5.3"},"label":"GLM-5.3"}]}}}`,
		"provider-only-entry": `{"settings":{"model":{"current":{"providerId":"zai-api","modelId":"GLM-5.3"},` +
			`"available":[{"ref":{"providerId":"zai-api"},"label":"broken"}]}}}`,
		"model-only-entry": `{"settings":{"model":{"current":{"providerId":"zai-api","modelId":"GLM-5.3"},` +
			`"available":[{"ref":{"modelId":"GLM-5.3"},"label":"broken"}]}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			settings, ok := acpfront.ParseSettings(json.RawMessage(raw))
			if !ok {
				t.Fatal("valid JSON settings envelope must parse")
			}
			if settings.Model.Current != nil && (settings.Model.Current.ProviderID == "" ||
				settings.Model.Current.ModelID == "") {
				t.Fatalf("partial current ref was accepted: %+v", settings.Model.Current)
			}
			for _, entry := range settings.Model.Available {
				if entry.Ref.ProviderID == "" || entry.Ref.ModelID == "" {
					t.Fatalf("partial available ref was accepted: %+v", entry.Ref)
				}
			}
			for _, option := range acpfront.BuildConfigOptions(settings) {
				if option.CurrentValue == "" || strings.HasPrefix(option.CurrentValue, "/") ||
					strings.HasSuffix(option.CurrentValue, "/") {
					t.Fatalf("partial ref reached rendered currentValue %q", option.CurrentValue)
				}
			}
		})
	}
}

// assertSelectOptions compares the emitted select entries element-for-
// element against the fixture-derived expectation (the verbatim-mirror
// discipline: order, values, labels, descriptions — an empty Description
// stays empty, never synthesized).
func assertSelectOptions(t *testing.T, got, want []acpport.SelectOptionDef) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("select options = %+v, want %d entries: %+v", got, len(want), want)
	}

	for i, w := range want {
		g := got[i]
		if g.Value != w.Value || g.Name != w.Name || g.Description != w.Description {
			t.Fatalf("select options[%d] = %+v, want %+v (verbatim-mirror violation — D-08)", i, g, w)
		}
	}
}
