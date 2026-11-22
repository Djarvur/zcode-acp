package interact_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/interact"
)

// TestFamilySelections locks the production entitlement gate (AUTH-02,
// plan 06-05) arm by arm — 06-RESEARCH.md Pattern 2's entitlement rule over
// READ-ONLY desktop state:
//
//	(a) the entitled arm — selections[family].kind == individual-coding-plan,
//	    modes[family] != apiKey, the store carries oauth:<family>:
//	    access_token → TRUE;
//	(b) mode == apiKey vetoes;
//	(c) a different selection kind (the start-plan family of the live
//	    probe) vetoes;
//	(d) a missing store token vetoes;
//	(e) a missing setting.json answers false with a structural log (a fresh
//	    machine is simply not logged in);
//	(f) an undecodable setting.json answers false through the same honest
//	    bool+log posture — never a panic;
//	(g) an ABSENT mode entry does not veto (the verified rule's !== apiKey
//	    strict-inequality semantics — the mode gate only rejects the
//	    explicit apiKey value).
//
// The gate composes the 06-01 store reader (no parallel read path) and
// every log line carries booleans/codes only — no value material.
func TestFamilySelections(t *testing.T) {
	t.Run("entitled arm answers true", testFamselEntitled)
	t.Run("apiKey mode vetoes entitlement", testFamselAPIKeyMode)
	t.Run("a different selection kind vetoes entitlement", testFamselOtherKind)
	t.Run("missing store token vetoes entitlement", testFamselNoToken)
	t.Run("missing setting.json answers false with a structural log", testFamselMissingSetting)
	t.Run("undecodable setting.json answers false honestly", testFamselUndecodableSetting)
	t.Run("absent mode entry does not veto", testFamselAbsentMode)
}

// famselSetting is the live-probed setting.json shape (06-RESEARCH.md
// Pattern 2's V-EXP probe: selections carry kind objects; modes carry
// plain strings) with the fixture families zai (coding plan) and bigmodel
// (start plan).
func famselSetting(selections, modes map[string]any) string {
	doc, err := json.Marshal(map[string]any{
		"providerFamilyConnectionSelections": selections,
		"modelProviderFamilyModes":           modes,
	})
	if err != nil {
		panic(err) // fixture construction — unreachable for map inputs
	}

	return string(doc)
}

// famselFixtureSetting writes the fixture setting.json and returns its path.
func famselFixtureSetting(t *testing.T, doc string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "setting.json")
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write setting fixture: %v", err)
	}

	return path
}

// famselFixtureStore writes the fixture credentials.json (synthetic marker
// material only) and returns the 06-01 reader over it — the SAME reader the
// gate composes in production (no parallel read path).
func famselFixtureStore(t *testing.T, entries map[string]string) *interact.CredentialStore {
	t.Helper()

	path := filepath.Join(t.TempDir(), "credentials.json")

	doc, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal credentials fixture: %v", err)
	}

	if err := os.WriteFile(path, doc, 0o600); err != nil {
		t.Fatalf("write credentials fixture: %v", err)
	}

	// v2 loader nil: the fixture entries are plaintext (the store's default
	// dispatch arm — the reader never touches a keyring for them).
	return interact.NewCredentialStore(path, nil)
}

// famselCaptureLog swaps the default slog handler for a capture buffer and
// restores it on cleanup.
func famselCaptureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return &buf
}

// famselZaiEntitledSelections is the fixture's entitled-arm selections map.
func famselZaiEntitledSelections() map[string]any {
	return map[string]any{
		"zai":      map[string]any{"kind": "individual-coding-plan"},
		"bigmodel": map[string]any{"kind": "start-plan"},
	}
}

// famselOAuthModes is the fixture's modes map (both families on oauth — the
// live-probed values).
func famselOAuthModes() map[string]any {
	return map[string]any{"zai": "oauth", "bigmodel": "oauth"}
}

// testFamselEntitled is arm (a): kind + mode + store token all hold — the
// gate answers true.
func testFamselEntitled(t *testing.T) {
	setting := famselFixtureSetting(t, famselSetting(famselZaiEntitledSelections(), famselOAuthModes()))
	store := famselFixtureStore(t, map[string]string{
		"oauth:zai:access_token": "fixture-synthetic-not-a-real-login-token",
	})

	gate := interact.NewEntitlementGate(setting, store)

	if !gate.Entitled(context.Background(), "zai") {
		t.Fatal("gate answered false for the fully-entitled arm — " +
			"want true (kind + non-apiKey mode + store token)")
	}
}

// testFamselAPIKeyMode is arm (b): an explicit apiKey mode vetoes even with
// kind + token holding.
func testFamselAPIKeyMode(t *testing.T) {
	setting := famselFixtureSetting(t, famselSetting(famselZaiEntitledSelections(),
		map[string]any{"zai": "apiKey", "bigmodel": "oauth"}))
	store := famselFixtureStore(t, map[string]string{
		"oauth:zai:access_token": "fixture-synthetic-not-a-real-login-token",
	})

	gate := interact.NewEntitlementGate(setting, store)

	if gate.Entitled(context.Background(), "zai") {
		t.Fatal("gate answered true with mode == apiKey — the explicit apiKey mode must veto")
	}
}

// testFamselOtherKind is arm (c): a selection kind other than
// individual-coding-plan (the start-plan family of the live probe) vetoes
// even with the store carrying a token.
func testFamselOtherKind(t *testing.T) {
	setting := famselFixtureSetting(t, famselSetting(famselZaiEntitledSelections(), famselOAuthModes()))
	store := famselFixtureStore(t, map[string]string{
		"oauth:bigmodel:access_token": "fixture-synthetic-not-a-real-login-token",
	})

	gate := interact.NewEntitlementGate(setting, store)

	if gate.Entitled(context.Background(), "bigmodel") {
		t.Fatal("gate answered true for the start-plan kind — only individual-coding-plan entitles")
	}
}

// testFamselNoToken is arm (d): the store holds no oauth token for the
// family — false even with kind + mode holding.
func testFamselNoToken(t *testing.T) {
	setting := famselFixtureSetting(t, famselSetting(famselZaiEntitledSelections(), famselOAuthModes()))
	store := famselFixtureStore(t, map[string]string{}) // fresh store: no entries

	gate := interact.NewEntitlementGate(setting, store)

	if gate.Entitled(context.Background(), "zai") {
		t.Fatal("gate answered true without a store token — the desktop login must hold")
	}
}

// testFamselMissingSetting is arm (e): the setting.json is absent (a fresh
// machine) — false with a structural log carrying the class code, never a
// value.
func testFamselMissingSetting(t *testing.T) {
	buf := famselCaptureLog(t)

	missing := filepath.Join(t.TempDir(), "absent", "setting.json")
	store := famselFixtureStore(t, map[string]string{
		"oauth:zai:access_token": "fixture-synthetic-not-a-real-login-token",
	})

	gate := interact.NewEntitlementGate(missing, store)

	if gate.Entitled(context.Background(), "zai") {
		t.Fatal("gate answered true with the setting.json absent — a fresh machine is not entitled")
	}

	logged := buf.String()
	if !strings.Contains(logged, "setting-missing") {
		t.Fatalf("missing-setting degradation logged %q — want the structural setting-missing class", logged)
	}

	if strings.Contains(logged, "fixture-synthetic-not-a-real-login-token") {
		t.Fatal("the log carries credential material — logs carry booleans/codes only (T-06-15)")
	}
}

// testFamselUndecodableSetting is arm (f): the setting.json is undecodable
// — the same honest false (coded structural log, never a panic).
func testFamselUndecodableSetting(t *testing.T) {
	buf := famselCaptureLog(t)

	setting := famselFixtureSetting(t, `{"providerFamilyConnectionSelections": NOT-JSON`)
	store := famselFixtureStore(t, map[string]string{
		"oauth:zai:access_token": "fixture-synthetic-not-a-real-login-token",
	})

	gate := interact.NewEntitlementGate(setting, store)

	if gate.Entitled(context.Background(), "zai") {
		t.Fatal("gate answered true with an undecodable setting.json — honest degradation must answer false")
	}

	logged := buf.String()
	if !strings.Contains(logged, "setting-undecodable") {
		t.Fatalf("undecodable-setting degradation logged %q — want the structural setting-undecodable class", logged)
	}

	if strings.Contains(logged, "NOT-JSON") {
		t.Fatal("the log carries the file dump — logs carry booleans/codes only (T-06-15)")
	}
}

// testFamselAbsentMode is arm (g): the verified rule is a strict
// inequality against the explicit apiKey value — an ABSENT modes entry does
// not veto (kind + token holding → true).
func testFamselAbsentMode(t *testing.T) {
	setting := famselFixtureSetting(t, famselSetting(famselZaiEntitledSelections(),
		map[string]any{"bigmodel": "oauth"})) // no zai entry
	store := famselFixtureStore(t, map[string]string{
		"oauth:zai:access_token": "fixture-synthetic-not-a-real-login-token",
	})

	gate := interact.NewEntitlementGate(setting, store)

	if !gate.Entitled(context.Background(), "zai") {
		t.Fatal("gate vetoed on an absent mode entry — the rule rejects only the explicit apiKey value")
	}
}
