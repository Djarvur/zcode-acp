package acpfront

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/core"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
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

func catalogLabelKey() string  { return "label" }
func catalogHighValue() string { return string([]byte{'h', 'i', 'g', 'h'}) }
func catalogLowValue() string  { return "low" }
func catalogHighName() string  { return "High" }
func catalogLowName() string   { return "Low" }

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
		entries = append(entries, map[string]any{"ref": ref(id), catalogLabelKey(): id})
	}
	raw, err := json.Marshal(map[string]any{"settings": map[string]any{"model": map[string]any{
		"current": ref(current), "lastUsed": ref(lastUsed), "available": entries,
	}}})
	if err != nil {
		panic(err) // fixture contains only JSON-safe scalar/map/slice values
	}

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
		//nolint:lll // each fixture row keeps its input snapshots and expected result together
		{"preserve listed current", catalogSnapshot("new/B", "", "old/A", "new/B"), catalogSnapshot("old/A", "", "old/A"), true, "old/A", 1},
		//nolint:lll // each fixture row keeps its input snapshots and expected result together
		{"probe current requires live setModel", catalogSnapshot("new/B", "", "new/B", "new/C"), catalogSnapshot("old/A", "", "old/A"), false, "", 0},
		{"listed lastUsed cannot replace disappeared current", catalogSnapshot("new/B", "", "new/B", "new/C"),
			catalogSnapshot("old/A", "new/B", "old/A", "new/B"), false, "", 0},
		//nolint:lll // each fixture row keeps its input snapshots and expected result together
		{"preserve without probe current", catalogSnapshot("", "", "old/A", "new/B"), catalogSnapshot("old/A", "", "old/A"), true, "old/A", 1},
		{"no fallback", catalogSnapshot("", "", "new/B"), catalogSnapshot("old/A", "", "old/A"), false, "", 0},
		//nolint:lll // each fixture row keeps its input snapshots and expected result together
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

//nolint:funlen // both close arms form one deterministic publication/teardown contract
func TestCatalogPublicationCloseGate(t *testing.T) {
	t.Run("closed session", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, interleaveScenario(0))
		defer closeLegs()
		openInterleaveSession(t, f, t.TempDir())
		f.recordSettings(interleaveSID, catalogSnapshot("old/A", "", "old/A"))
		calls := 0
		f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
			calls++

			return nil
		})
		if err := f.CloseSession(context.Background(), interleaveSID); err != nil {
			t.Fatalf("close: %v", err)
		}
		merged := catalogSnapshot("new/B", "", "new/B")
		if f.publishCatalogUpdate(context.Background(), interleaveSID, merged,
			BuildConfigOptions(mustSettings(t, merged))) {
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
			once.Do(func() {
				close(entered)
				<-release
			})

			return nil
		})
		merged := catalogSnapshot("new/B", "", "new/B")
		published := make(chan bool, 1)
		options := BuildConfigOptions(mustSettings(t, merged))
		go func() {
			published <- f.publishCatalogUpdate(context.Background(), interleaveSID, merged, options)
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

var errCatalogWrite = errors.New("write failed")

func TestCatalogPublicationFailureThenIdenticalRefresh(t *testing.T) {
	const sid = "sess_retry"
	f := New(nil)
	f.registry.Register(core.Session{ID: sid})
	calls := 0
	f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
		calls++
		if calls == 1 {
			return errCatalogWrite
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

func TestCatalogWorkerFailureThenIdenticalRefresh(t *testing.T) {
	probe := catalogSnapshot("new/B", "", "new/B")
	probe = json.RawMessage(`{"session":{"sessionId":"sess_probe"},` + string(probe[1:]))
	sc := &fake.Scenario{
		Name: "catalog-worker-retry",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodProviderRefreshRegistry, Times: 2,
				Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
			{Method: zcode.MethodSessionCreate, Times: 2, Reply: &fake.Reply{Result: probe}},
			{Method: zcode.MethodSessionClose, Times: 2,
				Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	const sid = interleaveSID
	f.recordSettings(sid, json.RawMessage(strings.Replace(string(catalogSnapshot("new/B", "", "new/B")),
		`"label":"new/B"`, `"label":"Old B"`, 1)))
	calls := 0
	f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
		calls++
		if calls == 1 {
			return errCatalogWrite
		}

		return nil
	})
	const signal = "identical-registry-signal"
	f.RequestCatalogRefresh(context.Background())
	awaitCatalogWorker(t, f)
	f.catalogRefreshMu.Lock()
	remembered := f.catalogRefreshSignal
	f.catalogRefreshMu.Unlock()
	if remembered != "" {
		t.Fatalf("failed pass retained dedupe signal %q", remembered)
	}
	f.requestCatalogRefresh(context.Background(), signal)
	awaitCatalogWorker(t, f)
	if calls != 2 {
		t.Fatalf("sink calls = %d, want failed pass plus identical retry", calls)
	}
	if got, _ := f.latestSettings(sid); !reflect.DeepEqual(mustSettings(t, got), mustSettings(t, probe)) {
		t.Fatalf("retry did not record confirmed catalog: %s", got)
	}
}

func awaitCatalogWorker(t *testing.T, f *Front) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		f.catalogRefreshMu.Lock()
		running := f.catalogRefreshRunning
		f.catalogRefreshMu.Unlock()
		if !running {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("catalog worker did not finish within the bound")
}

func TestCatalogObserverDoesNotTakeClosingLock(t *testing.T) {
	f, closeLegs := newInterleaveFront(t, interleaveScenario(0))
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	f.recordSettings(interleaveSID, catalogSnapshot("old/A", "", "old/A"))

	f.closingMu.Lock()
	locked := true
	defer func() {
		if locked {
			f.closingMu.Unlock()
		}
	}()
	done := make(chan struct{})
	go func() {
		f.observeEnvelope(zcode.Envelope{Method: zcode.MethodStateUpdated,
			Params: json.RawMessage(`{"patch":{"providers":{"zai-api":{"models":["B"]}}}}`)})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("registry observer blocked on closingMu held by close/resume")
	}
	f.closingMu.Unlock()
	locked = false
	awaitCatalogWorker(t, f)
}

func TestCatalogPublicationSerializesModelMutation(t *testing.T) {
	sc := &fake.Scenario{
		Name: "catalog-publication-mutation",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{Result: json.RawMessage(
				`{"settings":{"model":{"current":{"providerId":"new","modelId":"C"},` +
					`"available":[{"ref":{"providerId":"new","modelId":"C"},"label":"C"}]}}}`)}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	const sid = interleaveSID
	f.recordSettings(sid, catalogSnapshot("old/A", "", "old/A", "new/B", "new/C"))
	entered := make(chan struct{})
	release := make(chan struct{})
	f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
		close(entered)
		<-release

		return nil
	})
	merged := catalogSnapshot("new/B", "", "old/A", "new/B", "new/C")
	published := make(chan bool, 1)
	go func() {
		published <- f.publishCatalogUpdate(context.Background(), sid, merged,
			BuildConfigOptions(mustSettings(t, merged)))
	}()
	<-entered
	mutated := make(chan error, 1)
	go func() {
		_, err := f.SetSessionConfigOption(context.Background(), sid, optionIDModel, "old/A")
		mutated <- err
	}()
	select {
	case err := <-mutated:
		t.Fatalf("mutation completed during catalog client I/O: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if !<-published {
		t.Fatal("catalog publication failed")
	}
	if err := <-mutated; err != nil {
		t.Fatalf("model mutation: %v", err)
	}
	latest, _ := f.latestSettings(sid)
	settings := mustSettings(t, latest)
	if settings.Model.Current == nil || settings.Model.Current.ModelID != "C" {
		t.Fatalf("older catalog snapshot overwrote newer mutation: %s", latest)
	}
}

func catalogProbeWithThought(current, level string) json.RawMessage {
	raw := catalogSnapshot(current, "", "new/B", "new/C")
	raw = json.RawMessage(strings.Replace(string(raw), `"label":"new/B"`,
		//nolint:lll // raw reasoning fixture stays adjacent to its label replacement
		`"label":"B","reasoning":{"defaultLevel":"low","levels":[{"value":"high","label":"High"},{"value":"low","label":"Low"}]}`, 1))
	var root map[string]any
	_ = json.Unmarshal(raw, &root)
	settings, ok := root["settings"].(map[string]any)
	if !ok {
		panic("fixture settings missing")
	}
	settings["thoughtLevel"] = map[string]any{"enabled": true, "current": level,
		"available": []any{map[string]any{"value": catalogHighValue(), catalogLabelKey(): catalogHighName()},
			map[string]any{"value": catalogLowValue(), catalogLabelKey(): catalogLowName()}}}
	out, err := json.Marshal(root)
	if err != nil {
		panic(err)
	}

	return json.RawMessage(`{"session":{"sessionId":"sess_probe"},` + string(out[1:]))
}

func TestCatalogFallbackUsesLiveSetModelConfirmation(t *testing.T) {
	probe := catalogProbeWithThought("new/B", "low")
	confirmed := catalogProbeWithThought("new/B", "high")
	confirmed = json.RawMessage(strings.Replace(string(confirmed),
		`"sessionId":"sess_probe"`, `"sessionId":"sess_live"`, 1))
	sc := &fake.Scenario{
		Name: "catalog-live-fallback",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodProviderRefreshRegistry,
				Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
			{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: probe}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
			{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{Result: confirmed}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	openInterleaveSession(t, f, t.TempDir())
	const sid = interleaveSID
	previous := catalogProbeWithThought("old/A", "high")
	f.recordSettings(sid, previous)
	var notes []acpport.SessionNote
	f.BindUpdateSink(func(_ context.Context, note acpport.SessionNote) error {
		notes = append(notes, note)

		return nil
	})
	f.RequestCatalogRefresh(context.Background())
	awaitCatalogWorker(t, f)
	if len(notes) != 1 {
		t.Fatalf("catalog notes = %d, want 1 after live fallback confirmation", len(notes))
	}
	if len(notes[0].ConfigOptions) != 2 || notes[0].ConfigOptions[1].CurrentValue != "high" {
		t.Fatalf("fallback did not use live reply reasoning level: %+v", notes[0].ConfigOptions)
	}
	tr := closeLegs()
	setAt := catalogTranscriptNthMethod(tr, zcode.MethodSessionSetModel, 1)
	closeAt := catalogTranscriptNthMethod(tr, zcode.MethodSessionClose, 1)
	if setAt < 0 || closeAt < 0 || setAt <= closeAt {
		t.Fatalf("backend order setModel=%d probe-close=%d; "+
			"live setModel must follow probe close and precede publication", setAt, closeAt)
	}
}

func TestCatalogFallbackCommitsCurrentWithoutInvalidReasoning(t *testing.T) {
	for name, confirmed := range map[string]json.RawMessage{
		"invalid reasoning": catalogProbeWithThought("new/B", "medium"),
		"absent reasoning": withoutSettingsSection(
			catalogProbeWithThought("new/B", catalogHighValue()), "thoughtLevel"),
	} {
		t.Run(name, func(t *testing.T) {
			sc := &fake.Scenario{
				Name: "catalog-fallback-no-reasoning",
				OnRequest: []fake.RequestRule{
					{Method: zcode.MethodProviderRefreshRegistry,
						Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
					{Method: zcode.MethodSessionCreate,
						Reply: &fake.Reply{Result: catalogProbeWithThought("new/B", catalogLowValue())}},
					{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
					{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{Result: confirmed}},
				},
				EOFDrainMs: 1,
			}
			f, closeLegs := newInterleaveFront(t, sc)
			defer closeLegs()
			openInterleaveSession(t, f, t.TempDir())
			f.recordSettings(interleaveSID, catalogProbeWithThought("old/A", catalogHighValue()))
			var notes []acpport.SessionNote
			f.BindUpdateSink(func(_ context.Context, note acpport.SessionNote) error {
				notes = append(notes, note)

				return nil
			})
			f.RequestCatalogRefresh(context.Background())
			awaitCatalogWorker(t, f)
			if len(notes) != 1 || len(notes[0].ConfigOptions) != 1 {
				t.Fatalf("fallback notes = %+v, want model-only publication", notes)
			}
			if notes[0].ConfigOptions[0].CurrentValue != "new/B" {
				t.Fatalf("fallback current = %q, want new/B", notes[0].ConfigOptions[0].CurrentValue)
			}
			latest, _ := f.latestSettings(interleaveSID)
			settings := mustSettings(t, latest)
			if settings.Model.Current == nil || settings.Model.Current.ModelID != "B" ||
				settings.ThoughtLevel.Enabled || settings.ThoughtLevel.Current != "" {
				t.Fatalf("valid current was not committed with reasoning omitted: %s", latest)
			}
		})
	}
}

func TestCatalogFallbackClosesOnUnreconciledReply(t *testing.T) {
	for name, confirmed := range map[string]json.RawMessage{
		"wrong current":      catalogProbeWithThought("new/C", catalogHighValue()),
		"malformed settings": json.RawMessage(`{"settings":"bad"}`),
	} {
		t.Run(name, func(t *testing.T) {
			sc := &fake.Scenario{
				Name: "catalog-fallback-mismatch",
				OnRequest: []fake.RequestRule{
					{Method: zcode.MethodProviderRefreshRegistry,
						Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
					{Method: zcode.MethodSessionCreate,
						Reply: &fake.Reply{Result: catalogProbeWithThought("new/B", catalogLowValue())}},
					{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
					{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{Result: confirmed}},
				},
				EOFDrainMs: 1,
			}
			f, closeLegs := newInterleaveFront(t, sc)
			defer closeLegs()
			openInterleaveSession(t, f, t.TempDir())
			f.recordSettings(interleaveSID, catalogProbeWithThought("old/A", catalogHighValue()))
			calls := 0
			f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
				calls++

				return nil
			})
			f.RequestCatalogRefresh(context.Background())
			awaitCatalogWorker(t, f)
			if calls != 0 {
				t.Fatalf("unreconciled fallback published %d updates", calls)
			}
			if _, ok := f.latestSettings(interleaveSID); ok {
				t.Fatal("unreconciled successful setModel retained known-stale settings")
			}
			if _, err := f.registry.Get(interleaveSID); err == nil {
				t.Fatal("unreconciled successful setModel did not close the session")
			}
		})
	}
}

func TestCatalogFailedFallbackRetriesAfterHealthyFanout(t *testing.T) {
	probe := catalogProbeWithThought("new/B", catalogLowValue())
	confirmed := catalogProbeWithThought("new/B", catalogHighValue())
	sc := &fake.Scenario{
		Name: "catalog-fallback-retry",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodProviderRefreshRegistry, Times: 2,
				Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
			{Method: zcode.MethodSessionCreate, Times: 2, Reply: &fake.Reply{Result: probe}},
			{Method: zcode.MethodSessionClose, Times: 2,
				Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
			{Method: zcode.MethodSessionSetModel, Times: 1,
				Reply: &fake.Reply{Error: &zcode.WireError{Code: -32603, Message: "transient"}}},
			{Method: zcode.MethodSessionSetModel, Times: 1, Reply: &fake.Reply{Result: confirmed}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	f.registry.Register(core.Session{ID: "z-healthy"})
	f.recordSettings(interleaveSID, catalogProbeWithThought("old/A", catalogHighValue()))
	healthyOld := json.RawMessage(strings.Replace(string(catalogProbeWithThought("new/B", catalogHighValue())),
		`"label":"B"`, `"label":"Old B"`, 1))
	f.recordSettings("z-healthy", healthyOld)
	var ids []string
	f.BindUpdateSink(func(_ context.Context, note acpport.SessionNote) error {
		ids = append(ids, note.SessionID)

		return nil
	})
	const signal = "fallback-retry-signal"
	f.requestCatalogRefresh(context.Background(), signal)
	awaitCatalogWorker(t, f)
	if len(ids) != 1 || ids[0] != "z-healthy" {
		t.Fatalf("failed fallback prevented healthy fan-out: %v", ids)
	}
	f.catalogRefreshMu.Lock()
	remembered := f.catalogRefreshSignal
	f.catalogRefreshMu.Unlock()
	if remembered != "" {
		t.Fatalf("failed fallback retained dedupe signal %q", remembered)
	}
	f.requestCatalogRefresh(context.Background(), signal)
	awaitCatalogWorker(t, f)
	if len(ids) != 2 || ids[1] != interleaveSID {
		t.Fatalf("identical retry did not publish recovered fallback: %v", ids)
	}
	tr := closeLegs()
	if got := catalogTranscriptMethodCount(tr, zcode.MethodProviderRefreshRegistry); got != 2 {
		t.Fatalf("identical fallback retry issued %d refreshes, want 2", got)
	}
}

func TestCatalogNoCurrentProbePreservesListedSelection(t *testing.T) {
	probe := catalogSnapshot("", "", "old/A", "new/B")
	probe = json.RawMessage(`{"session":{"sessionId":"sess_probe"},` + string(probe[1:]))
	sc := &fake.Scenario{
		Name: "catalog-no-current-probe",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodProviderRefreshRegistry,
				Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
			{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: probe}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	f.recordSettings(interleaveSID, catalogSnapshot("old/A", "", "old/A"))
	calls := 0
	f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
		calls++

		return nil
	})
	f.RequestCatalogRefresh(context.Background())
	awaitCatalogWorker(t, f)
	if calls != 1 {
		t.Fatalf("no-current probe suppressed listed prior selection: calls=%d", calls)
	}
}

func TestCatalogPendingSignalSurvivesFailedPass(t *testing.T) {
	probe := catalogSnapshot("new/B", "", "new/B")
	probe = json.RawMessage(`{"session":{"sessionId":"sess_probe"},` + string(probe[1:]))
	sc := &fake.Scenario{
		Name: "catalog-pending-signal",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodProviderRefreshRegistry, Times: 2, DelayMs: 30,
				Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
			{Method: zcode.MethodSessionCreate, Times: 2, Reply: &fake.Reply{Result: probe}},
			{Method: zcode.MethodSessionClose, Times: 2,
				Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	openInterleaveSession(t, f, t.TempDir())
	previous := json.RawMessage(strings.Replace(string(catalogSnapshot("new/B", "", "new/B")),
		`"label":"new/B"`, `"label":"Old B"`, 1))
	f.recordSettings(interleaveSID, previous)
	entered := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
		calls++
		if calls == 1 {
			close(entered)
			<-release

			return errCatalogWrite
		}

		return nil
	})
	f.requestCatalogRefresh(context.Background(), "signal-A")
	<-entered
	f.requestCatalogRefresh(context.Background(), "signal-B")
	close(release)
	awaitCatalogWorker(t, f)
	if calls != 2 {
		t.Fatalf("sink calls = %d, want failed A plus pending B", calls)
	}
	f.catalogRefreshMu.Lock()
	current := f.catalogRefreshSignal
	f.catalogRefreshMu.Unlock()
	if current != "signal-B" {
		t.Fatalf("pending signal was not installed: %q", current)
	}
	f.requestCatalogRefresh(context.Background(), "signal-B")
	time.Sleep(30 * time.Millisecond)
	tr := closeLegs()
	if got := catalogTranscriptMethodCount(tr, zcode.MethodProviderRefreshRegistry); got != 2 {
		t.Fatalf("signal B repeated after its pass: refreshes=%d", got)
	}
}

func TestCatalogClosingSkipDoesNotAbortHealthyFanout(t *testing.T) {
	f, closeLegs := newInterleaveFront(t, interleaveScenario(0))
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	f.registry.Register(core.Session{ID: "healthy"})
	previous := json.RawMessage(strings.Replace(string(catalogSnapshot("new/B", "", "new/B")),
		`"label":"new/B"`, `"label":"Old B"`, 1))
	f.recordSettings(interleaveSID, previous)
	f.recordSettings("healthy", previous)
	f.closingMu.Lock()
	f.closing = map[string]bool{interleaveSID: true}
	f.closingMu.Unlock()
	var ids []string
	f.BindUpdateSink(func(_ context.Context, note acpport.SessionNote) error {
		ids = append(ids, note.SessionID)

		return nil
	})
	fresh := catalogSnapshot("new/B", "", "new/B")
	if err := f.publishFreshCatalog(context.Background(), fresh); err != nil {
		t.Fatalf("closing skip aborted healthy fan-out: %v", err)
	}
	if len(ids) != 1 || ids[0] != "healthy" {
		t.Fatalf("published sessions = %v, want only healthy", ids)
	}
}

func TestCatalogCloseSerializesInFlightMutation(t *testing.T) {
	sc := &fake.Scenario{
		Name: "catalog-close-mutation",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionSetModel, DelayMs: 200, Reply: &fake.Reply{Result: json.RawMessage(
				`{"settings":{"model":{"current":{"providerId":"new","modelId":"C"},` +
					`"available":[{"ref":{"providerId":"new","modelId":"C"},"label":"C"}]}}}`)}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	const sid = interleaveSID
	f.recordSettings(sid, catalogSnapshot("old/A", "", "old/A", "new/C"))
	mutated := make(chan error, 1)
	go func() {
		_, err := f.SetSessionConfigOption(context.Background(), sid, optionIDModel, "old/A")
		mutated <- err
	}()
	time.Sleep(25 * time.Millisecond)
	closed := make(chan error, 1)
	go func() { closed <- f.CloseSession(context.Background(), sid) }()
	select {
	case err := <-closed:
		t.Fatalf("close completed during in-flight setModel: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if err := <-mutated; err != nil {
		t.Fatalf("setModel: %v", err)
	}
	if err := <-closed; err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, ok := f.latestSettings(sid); ok {
		t.Fatal("late setModel recreated settings after close")
	}
}

func TestCatalogResumeSerializesPublication(t *testing.T) {
	resume := json.RawMessage(`{"session":{"sessionId":"sess_stress"},` +
		string(catalogSnapshot("new/D", "", "new/D")[1:]))
	sc := &fake.Scenario{
		Name: "catalog-resume-publication",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionResume, Reply: &fake.Reply{Result: resume}},
			{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{Result: json.RawMessage(`{"events":[]}`)}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	resumeCwd := t.TempDir()
	const sid = interleaveSID
	f.recordSettings(sid, catalogSnapshot("old/A", "", "old/A"))
	entered := make(chan struct{})
	release := make(chan struct{})
	f.BindUpdateSink(func(context.Context, acpport.SessionNote) error {
		close(entered)
		<-release

		return nil
	})
	merged := catalogSnapshot("new/B", "", "new/B")
	published := make(chan bool, 1)
	go func() {
		published <- f.publishCatalogUpdate(context.Background(), sid, merged,
			BuildConfigOptions(mustSettings(t, merged)))
	}()
	<-entered
	resumed := make(chan error, 1)
	go func() {
		_, err := f.ResumeSession(context.Background(), sid, resumeCwd)
		resumed <- err
	}()
	select {
	case err := <-resumed:
		t.Fatalf("resume completed during catalog publication: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if !<-published {
		t.Fatal("catalog publication failed")
	}
	if err := <-resumed; err != nil {
		t.Fatalf("resume: %v", err)
	}
	latest, _ := f.latestSettings(sid)
	settings := mustSettings(t, latest)
	if settings.Model.Current == nil || settings.Model.Current.ModelID != "D" {
		t.Fatalf("stale publication overwrote resume snapshot: %s", latest)
	}
}

func TestPickerRefusalTeardownUnderSessionStripe(t *testing.T) {
	sc := &fake.Scenario{
		Name: "picker-refusal-under-stripe",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionSetModel,
				Reply: &fake.Reply{Error: &zcode.WireError{Code: -32603, Message: "refused"}}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		EOFDrainMs: 1,
	}
	f, closeLegs := newInterleaveFront(t, sc)
	defer closeLegs()
	openInterleaveSession(t, f, t.TempDir())
	f.recordSettings(interleaveSID, catalogSnapshot("old/A", "", "old/A", "new/B"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := f.SetSessionConfigOption(ctx, interleaveSID, optionIDModel, "new/B"); err == nil {
		t.Fatal("refused setModel unexpectedly succeeded")
	}
	if _, ok := f.latestSettings(interleaveSID); ok {
		t.Fatal("picker-refusal teardown retained settings")
	}
}

func TestCatalogSessionLockCancellationDoesNotWait(t *testing.T) {
	f := New(nil)
	const sid = "cancel-lock"
	unlock, err := f.lockCatalogSession(context.Background(), sid)
	if err != nil {
		t.Fatalf("initial lock: %v", err)
	}
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := make(chan error, 1)
	go func() {
		_, lockErr := f.lockCatalogSession(ctx, sid)
		result <- lockErr
	}()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled lock error = %v, want context.Canceled", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("cancelled context waited on the session stripe")
	}
}

func catalogTranscriptNthMethod(tr fake.Transcript, method string, nth int) int {
	seen := 0
	for i, line := range tr.Lines {
		var env struct {
			Method string `json:"method"`
		}
		if json.Unmarshal([]byte(line.JSON), &env) == nil && env.Method == method {
			seen++
			if seen == nth {
				return i
			}
		}
	}

	return -1
}

func catalogTranscriptMethodCount(tr fake.Transcript, method string) int {
	count := 0
	for _, line := range tr.Lines {
		var env struct {
			Method string `json:"method"`
		}
		if json.Unmarshal([]byte(line.JSON), &env) == nil && env.Method == method {
			count++
		}
	}

	return count
}
