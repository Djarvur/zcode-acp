package acpfront

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/core"
)

func TestRegistryCatalogPatchClassification(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{"providers registry", `{"patch":{"providers":{"zai-api":{"models":["GLM-5.3"]}}}}`, true},
		{"prompt status", `{"patch":{"prompt":{"status":"started"}}}`, false},
		{"registry without providers", `{"patch":{"config":{"revision":1}}}`, false},
		{"malformed", `{`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registryCatalogPatch(json.RawMessage(tt.raw)); got != tt.want {
				t.Fatalf("registryCatalogPatch(%s) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func catalogSnapshot(current, lastUsed string, available ...string) json.RawMessage {
	ref := func(id string) any {
		if id == "" {
			return nil
		}
		parts := strings.SplitN(id, "/", 2)
		return map[string]any{"providerId": parts[0], "modelId": parts[1]}
	}
	entries := make([]any, 0, len(available))
	for _, id := range available {
		entries = append(entries, map[string]any{"ref": ref(id), "label": id})
	}
	raw, _ := json.Marshal(map[string]any{"settings": map[string]any{"model": map[string]any{
		"current": ref(current), "lastUsed": ref(lastUsed), "available": entries,
	}}})
	return raw
}

func TestSnapshotForLiveCatalogSafety(t *testing.T) {
	tests := []struct {
		name          string
		fresh         json.RawMessage
		previous      json.RawMessage
		wantOK        bool
		wantCurrent   string
		wantOptionLen int
	}{
		{"preserve listed current", catalogSnapshot("new/B", "", "old/A", "new/B"), catalogSnapshot("old/A", "", "old/A"), true, "old/A", 1},
		{"confirmed listed fallback", catalogSnapshot("new/B", "", "new/B", "new/C"), catalogSnapshot("old/A", "", "old/A"), true, "new/B", 1},
		{"preserve without probe current", catalogSnapshot("", "", "old/A", "new/B"), catalogSnapshot("old/A", "", "old/A"), true, "old/A", 1},
		{"no fallback", catalogSnapshot("", "", "new/B"), catalogSnapshot("old/A", "", "old/A"), false, "", 0},
		{"unlisted fallback", catalogSnapshot("gone/X", "", "new/B"), catalogSnapshot("old/A", "", "old/A"), false, "", 0},
		{"malformed fresh", json.RawMessage(`{`), catalogSnapshot("old/A", "", "old/A"), false, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, ok := snapshotForLiveCatalog(tt.fresh, tt.previous)
			if ok != tt.wantOK {
				t.Fatalf("snapshotForLiveCatalog ok = %v, want %v (merged=%s)", ok, tt.wantOK, merged)
			}
			if !ok {
				return
			}
			settings, parsed := ParseSettings(merged)
			if !parsed || settings.Model.Current == nil {
				t.Fatalf("merged snapshot has no parsed current: %s", merged)
			}
			if got := settings.Model.Current.ProviderID + "/" + settings.Model.Current.ModelID; got != tt.wantCurrent {
				t.Fatalf("merged current = %s, want %s", got, tt.wantCurrent)
			}
			if got := len(BuildConfigOptions(settings)); got != tt.wantOptionLen {
				t.Fatalf("rendered options = %d, want %d", got, tt.wantOptionLen)
			}
		})
	}
}

func TestCatalogPublicationCloseGate(t *testing.T) {
	t.Run("closed session", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, interleaveScenario(0))
		defer closeLegs()
		openInterleaveSession(t, f, t.TempDir())
		f.recordSettings(interleaveSID, catalogSnapshot("old/A", "", "old/A"))
		calls := 0
		f.BindUpdateSink(func(context.Context, acpport.SessionNote) error { calls++; return nil })
		if err := f.CloseSession(context.Background(), interleaveSID); err != nil {
			t.Fatalf("close: %v", err)
		}
		merged := catalogSnapshot("new/B", "", "new/B")
		if f.publishCatalogUpdate(context.Background(), interleaveSID, merged, BuildConfigOptions(mustSettings(t, merged))) {
			t.Fatal("stale publication passed the closed-session gate")
		}
		if calls != 0 {
			t.Fatalf("sink calls = %d, want 0", calls)
		}
		if _, ok := f.latestSettings(interleaveSID); ok {
			t.Fatal("closed session regained settings")
		}
	})

	t.Run("close during bounded write", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, interleaveScenario(0))
		defer closeLegs()
		openInterleaveSession(t, f, t.TempDir())
		old := catalogSnapshot("old/A", "", "old/A")
		f.recordSettings(interleaveSID, old)
		entered := make(chan struct{})
		release := make(chan struct{})
		var once sync.Once
		calls := 0
		f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
			calls++
			once.Do(func() { close(entered); <-release })
			return nil
		})
		merged := catalogSnapshot("new/B", "", "new/B")
		published := make(chan bool, 1)
		go func() {
			published <- f.publishCatalogUpdate(context.Background(), interleaveSID, merged, BuildConfigOptions(mustSettings(t, merged)))
		}()
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("publication never entered sink")
		}
		if f.closingMu.TryLock() {
			f.closingMu.Unlock()
			t.Fatal("close gate was available during client I/O")
		}
		closed := make(chan error, 1)
		go func() { closed <- f.CloseSession(context.Background(), interleaveSID) }()
		close(release)
		if !<-published {
			t.Fatal("publication failed before close could tear down")
		}
		if err := <-closed; err != nil {
			t.Fatalf("close: %v", err)
		}
		if calls != 1 {
			t.Fatalf("sink calls = %d, want 1", calls)
		}
		if _, ok := f.latestSettings(interleaveSID); ok {
			t.Fatal("close did not drop the published settings")
		}
	})
}

func mustSettings(t *testing.T, raw json.RawMessage) Settings {
	t.Helper()
	settings, ok := ParseSettings(raw)
	if !ok {
		t.Fatalf("parse fixture: %s", raw)
	}
	return settings
}

func TestCatalogPublicationFailureThenIdenticalRefresh(t *testing.T) {
	const sid = "sess_retry"
	f := New(nil)
	f.registry.Register(core.Session{ID: sid})
	calls := 0
	f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
		calls++
		if calls == 1 {
			return errors.New("write failed")
		}
		return nil
	})
	old := catalogSnapshot("old/A", "", "old/A")
	f.recordSettings(sid, old)
	merged := catalogSnapshot("new/B", "", "new/B")
	options := BuildConfigOptions(mustSettings(t, merged))
	if f.publishCatalogUpdate(context.Background(), sid, merged, options) {
		t.Fatal("failed write reported publication success")
	}
	if got, _ := f.latestSettings(sid); !bytes.Equal(got, old) {
		t.Fatalf("failed write replaced old settings: %s", got)
	}
	if !f.publishCatalogUpdate(context.Background(), sid, merged, options) {
		t.Fatal("identical refresh did not retry after failed write")
	}
	if calls != 2 {
		t.Fatalf("sink calls = %d, want 2", calls)
	}
	if got, _ := f.latestSettings(sid); !bytes.Equal(got, merged) {
		t.Fatalf("successful retry did not record merged snapshot: %s", got)
	}
}
