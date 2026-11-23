package texttask_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/texttask"
)

// writeProviderConfig writes a provider_config.json fixture under the
// test's temp dir and returns its path.
func writeProviderConfig(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "provider_config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write provider config %s: %v", path, err)
	}

	return path
}

// TestResolveSelectionSources covers the resolver's two sources: the
// configured passthrough and the provider_config default reader.
func TestResolveSelectionSources(t *testing.T) {
	t.Run("a configured selection passes through unchanged", func(t *testing.T) {
		configured := &acpfront.ModelSelection{ProviderID: testProviderPersonal, ModelID: testModelPersonal}

		got, err := texttask.ResolveSelection(configured, filepath.Join(t.TempDir(), "absent.json"))
		if err != nil {
			t.Fatalf("ResolveSelection(configured): %v", err)
		}

		if got != *configured {
			t.Fatalf("ResolveSelection = %+v, want the configured selection %+v", got, *configured)
		}

		if got.ReasoningLevel != "" {
			t.Fatalf("ResolveSelection level = %q, want empty (options omitted)", got.ReasoningLevel)
		}
	})

	t.Run("the provider_config default supplies ids and the reasoning level", func(t *testing.T) {
		path := writeProviderConfig(t, `{"config":{"defaultModelSelection":`+
			`{"providerId":"`+testProviderZaiAPI+`","modelId":"`+testModelFlash+`",`+
			`"options":{"reasoningLevel":"`+testLevelHigh+`"}}},`+
			`"schemaVersion":1}`)

		got, err := texttask.ResolveSelection(nil, path)
		if err != nil {
			t.Fatalf("ResolveSelection(default): %v", err)
		}

		want := acpfront.ModelSelection{
			ProviderID: testProviderZaiAPI, ModelID: testModelFlash, ReasoningLevel: testLevelHigh,
		}
		if got != want {
			t.Fatalf("ResolveSelection = %+v, want %+v", got, want)
		}
	})
}

// TestResolveSelectionErrors covers the resolver's sentinel classes and the
// never-echo-credentials rule.
func TestResolveSelectionErrors(t *testing.T) {
	t.Run("a missing file is ErrNoSelection naming the path and the fix", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "absent.json")

		_, err := texttask.ResolveSelection(nil, path)
		if !errors.Is(err, texttask.ErrNoSelection) {
			t.Fatalf("ResolveSelection(missing) error = %v, want ErrNoSelection", err)
		}

		if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "--model") {
			t.Fatalf("error %q must name the path and the --model fix", err)
		}
	})

	t.Run("undecodable JSON is ErrNoSelection", func(t *testing.T) {
		path := writeProviderConfig(t, `{"config":`)

		_, err := texttask.ResolveSelection(nil, path)
		if !errors.Is(err, texttask.ErrNoSelection) {
			t.Fatalf("ResolveSelection(undecodable) error = %v, want ErrNoSelection", err)
		}
	})

	t.Run("an empty provider or model id is ErrInvalidDefaultSelection", func(t *testing.T) {
		path := writeProviderConfig(t, `{"config":{"defaultModelSelection":`+
			`{"providerId":"`+testProviderZaiAPI+`","modelId":""}},`+
			`"providers":{"apiKey":"SECRET-MARKER"}}`)

		_, err := texttask.ResolveSelection(nil, path)
		if !errors.Is(err, texttask.ErrInvalidDefaultSelection) {
			t.Fatalf("ResolveSelection(empty id) error = %v, want ErrInvalidDefaultSelection", err)
		}

		if strings.Contains(err.Error(), "SECRET-MARKER") {
			t.Fatalf("error %q must never echo provider credential fields", err)
		}
	})
}
