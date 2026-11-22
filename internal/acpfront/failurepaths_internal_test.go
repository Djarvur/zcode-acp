package acpfront

// WR-04 + IN-04 (plan 07-01) in-package failure-path locks. The map-entry
// assertions read the front's unexported cwd/settings stores directly —
// the external-package suite (newsession_test.go) locks the same flows
// through the wire surfaces only. The flows ride the composed back leg
// (fake.Serve ⇄ zcode.Conn ⇄ Front) with the shared interleave harness;
// session/new is driven through the acpport params type (acpport does not
// import this package, so the in-package test's import stays cycle-free).
//
//	TestCreateFailurePathsLeaveNoMapEntries — the WR-07 bounding invariant
//	  holds on ALL post-create failure paths: the subscribe-refused
//	  create, the launch-model-refused create, and the subscribe-refused
//	  resume each leave ZERO cwd/settings entries behind (the leak arms —
//	  the RED target), and the happy path still records with the teardown
//	  still dropping (the IN-04 liveness lock: the map is live, only the
//	  dead nil-guard is gone).

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	// failureSID is the scripted session id every arm attempts (the
	// create/resume snapshot digs it back out unchanged).
	failureSID = "sess_fp"

	// failureMethodNotFound is the unknown-method reply message the
	// scenario arms fail loud with (the corpus discipline: a request the
	// arm did not script must never get a silent answer).
	failureMethodNotFound = "MethodNotFound"

	// failureDrainMs is the EOF drain window (the arms terminate through
	// their request rules; the drain just bounds the serve exit).
	failureDrainMs = 1
)

// failureSnapshot is the create/resume reply every success arm answers
// with (the create-shaped snapshot carries the attempted id).
func failureSnapshot() json.RawMessage {
	return json.RawMessage(`{"session":{"sessionId":"` + failureSID + `"}}`)
}

// failureSubscribeReply is the subscribe success arm's reply.
const failureSubscribeReply = `{"sessionId":"sess_fp","eventSeq":0,"events":[]}`

// failurePathsScenario composes one arm's back leg from its rules; the
// unknown-method reply fails loud so a misrouted request can never pass
// silently (the corpus discipline).
func failurePathsScenario(rules ...fake.RequestRule) *fake.Scenario {
	return &fake.Scenario{
		Name:      "create-failure-paths",
		OnRequest: rules,
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: failureMethodNotFound},
		},
		EOFDrainMs: failureDrainMs,
	}
}

// assertNoEntryForSID requires the store to hold no entry for the attempted
// id — the WR-07 bounding invariant on the failure path under assertion.
func assertNoEntryForSID(t *testing.T, what string, store map[string]string) {
	t.Helper()

	if _, ok := store[failureSID]; ok {
		t.Fatalf("%s left a cwd entry for the attempted id behind (WR-07: the failure paths must not leak)", what)
	}
}

// assertNoSettingsForSID requires the settings store to hold no entry for
// the attempted id.
func assertNoSettingsForSID(t *testing.T, what string, store map[string]json.RawMessage) {
	t.Helper()

	if _, ok := store[failureSID]; ok {
		t.Fatalf("%s left a settings entry for the attempted id behind (WR-07: the failure paths must not leak)", what)
	}
}

// TestCreateFailurePathsLeaveNoMapEntries locks the WR-07 bounding
// invariant across all three post-create failure paths plus the IN-04
// liveness lock (the happy path still records; the teardown still drops —
// proving the stores are live and only the dead nil-guard was removed).
//
//nolint:funlen // each subtest is a full harness lock; splitting would scatter its sequencing
func TestCreateFailurePathsLeaveNoMapEntries(t *testing.T) {
	cwd := t.TempDir()
	ctx := context.Background()

	t.Run("subscribe-refused create leaves no cwd entry", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, failurePathsScenario(
			fake.RequestRule{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: failureSnapshot()}},
			fake.RequestRule{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{
				Error: &zcode.WireError{Code: zcode.CodeInternalError, Message: "subscribe refused"},
			}},
		))
		defer closeLegs()

		sid, _, err := f.NewSession(ctx, acpport.NewSessionParams{Cwd: cwd})
		if err == nil {
			t.Fatalf("session/new must fail when the subscribe leg refuses, got session %q", sid)
		}

		assertNoEntryForSID(t, "the subscribe-refused create", f.cwds)
	})

	t.Run("launch-model-refused create leaves no cwd and no settings entry", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, failurePathsScenario(
			fake.RequestRule{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: failureSnapshot()}},
			fake.RequestRule{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{
				Result: json.RawMessage(failureSubscribeReply),
			}},
			fake.RequestRule{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{
				Error: &zcode.WireError{Code: zcode.CodeInternalError, Message: "setModel refused"},
			}},
		))

		// The configured selection makes applyConfiguredModel fire the
		// launch setModel follow-up — the refusal arm under test.
		f.model = &ModelSelection{ProviderID: "zai-api", ModelID: "GLM-5.3-Flash", ReasoningLevel: "high"}

		defer closeLegs()

		sid, _, err := f.NewSession(ctx, acpport.NewSessionParams{Cwd: cwd})
		if err == nil {
			t.Fatalf("session/new must fail when the launch setModel refuses, got session %q", sid)
		}

		assertNoEntryForSID(t, "the launch-model-refused create", f.cwds)
		assertNoSettingsForSID(t, "the launch-model-refused create", f.settings)
	})

	t.Run("subscribe-refused resume leaves no cwd entry", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, failurePathsScenario(
			fake.RequestRule{Method: zcode.MethodSessionResume, Reply: &fake.Reply{Result: failureSnapshot()}},
			fake.RequestRule{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{
				Error: &zcode.WireError{Code: zcode.CodeInternalError, Message: "subscribe refused"},
			}},
		))
		defer closeLegs()

		if _, err := f.ResumeSession(ctx, failureSID, cwd); err == nil {
			t.Fatal("session/resume must fail when its re-issued subscribe refuses")
		}

		assertNoEntryForSID(t, "the subscribe-refused resume", f.cwds)
	})

	t.Run("the happy path still records and the teardown still drops (IN-04 liveness)", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, failurePathsScenario(
			fake.RequestRule{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: failureSnapshot()}},
			fake.RequestRule{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{
				Result: json.RawMessage(failureSubscribeReply),
			}},
			fake.RequestRule{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{Result: failureSnapshot()}},
			fake.RequestRule{Method: zcode.MethodSessionClose, Reply: &fake.Reply{
				Result: json.RawMessage(`{"closed":true}`),
			}},
		))
		defer closeLegs()

		sid, _, err := f.NewSession(ctx, acpport.NewSessionParams{Cwd: cwd})
		if err != nil {
			t.Fatalf("session/new: %v", err)
		}

		if sid != failureSID {
			t.Fatalf("session id = %q, want the scripted %q", sid, failureSID)
		}

		// The stores are LIVE: the happy path recorded both entries.
		if _, ok := f.cwds[failureSID]; !ok {
			t.Fatal("the happy path must record the session cwd (IN-04: removing the dead nil-guard must not strand the store)")
		}

		if _, ok := f.settings[failureSID]; !ok {
			t.Fatal("the happy path must record the confirmed settings (IN-04: removing the dead nil-guard must not strand the store)")
		}

		// The teardown still drops both — the map stays bounded with the
		// registry (the WR-07 discipline).
		if err := f.CloseSession(ctx, failureSID); err != nil {
			t.Fatalf("session/close: %v", err)
		}

		assertNoEntryForSID(t, "the teardown", f.cwds)
		assertNoSettingsForSID(t, "the teardown", f.settings)
	})
}
