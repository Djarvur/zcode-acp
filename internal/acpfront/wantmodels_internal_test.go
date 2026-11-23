package acpfront

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// wantListFixtureTable is a minimal revision-44 builtin table with THREE
// zhipu-account rules of ONE family: account:visible carries
// config.visibility "visible" and mode individual-coding-plan (the selected
// connection), account:team is visible too but carries the NON-selected
// team-coding-plan mode (the per-mode omission companion), and account:hidden
// carries "hidden" (the desktop's strict marker, compared exactly). The
// hidden row mirrors the real table's idle/highspeed class — same account
// family, its own model id.
const wantListFixtureTable = `{
  "revision": 44,
  "config": {
    "providerConfigRules": {
      "providerRules": [
        {
          "providerId": "account:visible",
          "providerName": "Visible Account",
          "config": {
            "group": "wantmodels-family",
            "builtinModelIds": ["M-A", "M-B"],
            "access": {"type": "zhipu-account", "mode": "individual-coding-plan", "accountType": "wantmodels-family"},
            "visibility": "visible"
          }
        },
        {
          "providerId": "account:team",
          "providerName": "Team Account",
          "config": {
            "group": "wantmodels-family",
            "builtinModelIds": ["M-Team"],
            "access": {"type": "zhipu-account", "mode": "team-coding-plan", "accountType": "wantmodels-family"},
            "visibility": "visible"
          }
        },
        {
          "providerId": "account:hidden",
          "providerName": "Hidden Account",
          "config": {
            "group": "wantmodels-family",
            "builtinModelIds": ["M-Hidden"],
            "access": {"type": "zhipu-account", "mode": "off-peak", "accountType": "wantmodels-family"},
            "visibility": "hidden"
          }
        }
      ]
    }
  }
}`

// The fixture's family and its selected connection mode (test-local
// spellings for the fixture table — no production catalog literal).
const (
	wantListFixtureFamily         = "wantmodels-family"
	wantListFixtureIndividualMode = "individual-coding-plan"
)

// TestWantModelsFromSnapshotOmitsHidden locks the read-back want list against
// visibility (D-08's verdict-side twin): a snapshot built from one visible
// and one hidden zhipu-account rule yields exactly the visible provider/model
// pairs (sorted), so the read-back watcher — and ProbeWantModel — can never
// target a hidden provider, which is why the catalog mirror (settings.model
// .available, pushed verbatim into configOptions) can never materialize one.
func TestWantModelsFromSnapshotOmitsHidden(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(wantListFixtureTable), 0o600); err != nil {
		t.Fatalf("write builtin fixture: %v", err)
	}

	snap, err := hostsvc.NewSnapshotBuilder(nil).BuildAccount(
		supervisor.Install{BuiltinFile: builtin}, func(string, string) bool { return true })
	if err != nil {
		t.Fatalf("build account snapshot: %v", err)
	}

	providers, _ := snap.Params["providers"].(map[string]any)

	// Captured before the assertions so the log always names what the
	// read-back watcher would see (the RED evidence quotes it).
	got := wantModelsFromSnapshot(snap)
	t.Logf("derived want list: %v", got)

	if _, ok := providers["account:hidden"]; ok {
		t.Fatalf("providers carries %q, want the config.visibility \"hidden\" rule omitted "+
			"(desktop-picker parity: the desktop's own model view filters the same field)", "account:hidden")
	}

	want := []string{"account:team/M-Team", "account:visible/M-A", "account:visible/M-B"}

	if !slices.Equal(got, want) {
		t.Fatalf("want list = %v, want exactly %v (sorted) — a hidden provider/model must never "+
			"reach the read-back watcher or the probe target", got, want)
	}

	if probe := ProbeWantModel(got); !slices.Contains(want, probe) {
		t.Fatalf("ProbeWantModel = %q, want one of the visible want entries %v — "+
			"never the hidden entry account:hidden/M-Hidden", probe, want)
	}
}

// TestWantModelsFromSnapshotOmitsNonSelectedMode locks the per-mode omission
// on the read-back side: with the fixture family selected on
// individual-coding-plan, the visible team-mode rule is omitted just like the
// hidden one — so its models can never enter the want list or become the
// ProbeWantModel target, and the verdict-side mirror still describes exactly
// what was pushed.
func TestWantModelsFromSnapshotOmitsNonSelectedMode(t *testing.T) {
	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(wantListFixtureTable), 0o600); err != nil {
		t.Fatalf("write builtin fixture: %v", err)
	}

	snap, err := hostsvc.NewSnapshotBuilder(nil).BuildAccount(
		supervisor.Install{BuiltinFile: builtin},
		func(family, mode string) bool {
			return family == wantListFixtureFamily && mode == wantListFixtureIndividualMode
		})
	if err != nil {
		t.Fatalf("build account snapshot: %v", err)
	}

	providers, _ := snap.Params["providers"].(map[string]any)

	got := wantModelsFromSnapshot(snap)
	t.Logf("derived want list: %v", got)

	for _, pid := range []string{"account:team", "account:hidden"} {
		if _, ok := providers[pid]; ok {
			t.Fatalf("providers carries %q, want the non-selected-mode/hidden rule omitted "+
				"(only the family's selected connection mode may be pushed)", pid)
		}
	}

	want := []string{"account:visible/M-A", "account:visible/M-B"}

	if !slices.Equal(got, want) {
		t.Fatalf("want list = %v, want exactly %v (sorted) — a non-selected-mode provider/model "+
			"must never reach the read-back watcher or the probe target", got, want)
	}

	if probe := ProbeWantModel(got); !slices.Contains(want, probe) {
		t.Fatalf("ProbeWantModel = %q, want one of the selected-mode want entries %v — "+
			"never the team entry account:team/M-Team", probe, want)
	}
}
