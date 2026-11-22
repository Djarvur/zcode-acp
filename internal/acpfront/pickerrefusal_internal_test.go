package acpfront

// WR-01 (plan 07-01) in-package picker-refusal teardown locks. The
// zombie-state assertions read the front's unexported registry and store
// state directly — the external-package suite (setconfigoption_test.go)
// locks the same flows through the client-visible surfaces only. The
// established session rides the composed back leg (fake.Serve ⇄
// zcode.Conn ⇄ Front) with the shared interleave harness; the picker ask
// is driven through the exported Front.SetSessionConfigOption (the same
// validated path the adapter's set_config_option arm takes).
//
//	TestPickerSetModelRefusalRunsFullTeardown — a refused backend setModel
//	  on an ESTABLISHED session must run the full SESS-01 teardown: the
//	  registry holds no entry for the id and the cwd/settings maps are
//	  empty (the zombie gone — the RED arm), the client still receives the
//	  coded setModel refusal (the reply is preserved), a follow-up prompt
//	  answers the closing/closed invalid-request class and never reaches
//	  the backend wire, and a SUCCESSFUL switch keeps the session fully
//	  alive (the teardown fires only on the refusal arm).

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	// pickerSID is the established session's scripted id (the create
	// snapshot digs it back out unchanged).
	pickerSID = "sess_wr01"

	// pickerFlashValue is the picker target (the providerId/modelId
	// serialization the snapshot's model option lists).
	pickerFlashValue = "zai-api/GLM-5.3-Flash"
)

// pickerNoopSink returns the follow-up prompt's update sink — the prompt
// under test must be rejected at the registry gate before any note could
// exist; a healthy session would fail this test loudly on the wire asserts
// long before the sink mattered.
func pickerNoopSink() acpport.UpdateSink {
	return func(_ context.Context, _ acpport.SessionNote) error {
		return nil
	}
}

// pickerSnapshot is the create reply: a settings snapshot whose model
// option lists the picker target — the validated ask resolves against it
// (the T-06-17 validate-first posture).
func pickerSnapshot() json.RawMessage {
	return json.RawMessage(`{
	  "session": {"sessionId": "` + pickerSID + `"},
	  "settings": {
	    "model": {
	      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
	      "available": [
	        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash"},
	        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
	      ]
	    }
	  }
	}`)
}

// pickerRefusalScenario scripts the established session's back leg: the
// create + subscribe preamble answers, the picker's setModel carries the
// refused arm, the teardown's session/close answers (the full SESS-01
// teardown issues one), and any session/send fails loud (a send after the
// refusal is exactly the zombie arm this lock forbids).
func pickerRefusalScenario(setModelRule fake.RequestRule) *fake.Scenario {
	return &fake.Scenario{
		Name: "picker-refusal-teardown",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: pickerSnapshot()}},
			{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{
				Result: json.RawMessage(`{"sessionId":"` + pickerSID + `","eventSeq":0,"events":[]}`),
			}},
			setModelRule,
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{
				Result: json.RawMessage(`{"closed":true}`),
			}},
		},
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: "MethodNotFound"},
		},
		EOFDrainMs: 1,
	}
}

// openPickerSession runs the create + subscribe preamble (no configured
// model — the picker's own setModel frames are the only mutation frames)
// and returns the established front + session id.
func openPickerSession(ctx context.Context, t *testing.T, f *Front, cwd string) string {
	t.Helper()

	sid, _, err := f.NewSession(ctx, acpport.NewSessionParams{Cwd: cwd})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if sid != pickerSID {
		t.Fatalf("session id = %q, want the scripted %q", sid, pickerSID)
	}

	return sid
}

// assertPickerZombieGone requires the post-refusal state to hold NO front
// session: the registry entry released, both stores dropped — the WR-01
// zombie (a registered session mutating a dead backend) is gone.
func assertPickerZombieGone(t *testing.T, f *Front, what string) {
	t.Helper()

	if _, getErr := f.registry.Get(pickerSID); getErr == nil {
		t.Fatalf("%s left the registry entry alive — the WR-01 zombie (later prompts send to a dead backend)", what)
	}

	if _, ok := f.cwds[pickerSID]; ok {
		t.Fatalf("%s left a cwd entry behind (the teardown must drop the stores)", what)
	}

	if _, ok := f.settings[pickerSID]; ok {
		t.Fatalf("%s left a settings entry behind (the teardown must drop the stores)", what)
	}
}

// assertPickerSendCount requires the back leg to record exactly want
// session/send frames — a follow-up prompt must never reach a torn-down
// (or refused) backend session.
func assertPickerSendCount(t *testing.T, tr fake.Transcript, want int) {
	t.Helper()

	sends := 0
	for _, line := range tr.Lines {
		if string(line.Dir) == "in" && strings.Contains(line.JSON, `"method":"session/send"`) {
			sends++
		}
	}

	if sends != want {
		t.Fatalf("back leg records %d session/send frames, want exactly %d "+
			"(a post-refusal prompt must never reach the backend)", sends, want)
	}
}

// TestPickerSetModelRefusalRunsFullTeardown locks the WR-01 contract: the
// refused picker setModel fully tears the front session down (no zombie),
// the refusal reply is preserved, the follow-up prompt lands the coded
// closing/closed class without a backend send, and the successful switch
// keeps the session fully alive.
//
//nolint:funlen,contextcheck // full harness locks; the harness owns its conn lifecycle (NewConn is ctx-free)
func TestPickerSetModelRefusalRunsFullTeardown(t *testing.T) {
	cwd := t.TempDir()
	ctx := context.Background()

	refusalRule := fake.RequestRule{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{
		Error: &zcode.WireError{Code: zcode.CodeInternalError, Message: "Reasoning level is required for this model"},
	}}

	t.Run("the refusal tears the front session down (no zombie)", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, pickerRefusalScenario(refusalRule))
		defer closeLegs()

		sid := openPickerSession(ctx, t, f, cwd)

		if _, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerFlashValue); err == nil {
			t.Fatal("a refused picker setModel must fail the ask, got success")
		}

		assertPickerZombieGone(t, f, "the refused picker setModel")
	})

	t.Run("the client still receives the coded setModel refusal", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, pickerRefusalScenario(refusalRule))
		defer closeLegs()

		sid := openPickerSession(ctx, t, f, cwd)

		_, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerFlashValue)
		if err == nil {
			t.Fatal("a refused picker setModel must fail the ask, got success")
		}

		// The reply is PRESERVED: the same coded internal class, unchanged
		// by the teardown (never swallowed, never replaced).
		var seamErr *acpport.Error
		if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInternal {
			t.Fatalf("refusal error = %v, want the coded internal setModel refusal (unchanged by the teardown)", err)
		}
	})

	t.Run("a follow-up prompt answers the closing/closed class, never a backend send", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, pickerRefusalScenario(refusalRule))

		sid := openPickerSession(ctx, t, f, cwd)

		if _, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerFlashValue); err == nil {
			t.Fatal("a refused picker setModel must fail the ask, got success")
		}

		_, promptErr := f.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: "after refusal"}, pickerNoopSink())
		if promptErr == nil {
			t.Fatal("a prompt for the torn-down session must fail, got success")
		}

		var seamErr *acpport.Error
		if !errors.As(promptErr, &seamErr) || seamErr.Code != acpport.CodeInvalidRequest {
			t.Fatalf("follow-up prompt error = %v, want the coded invalid-request closing/closed class", promptErr)
		}

		assertPickerSendCount(t, closeLegs(), 0)
	})

	t.Run("a successful picker setModel keeps the session fully alive", func(t *testing.T) {
		f, closeLegs := newInterleaveFront(t, pickerRefusalScenario(fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: pickerSnapshot()},
		}))
		defer closeLegs()

		sid := openPickerSession(ctx, t, f, cwd)

		if _, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerFlashValue); err != nil {
			t.Fatalf("the successful model switch must go through: %v", err)
		}

		// The teardown fires ONLY on the refusal arm: the registry entry is
		// alive and the confirmed store holds the reply's snapshot.
		if _, getErr := f.registry.Get(sid); getErr != nil {
			t.Fatalf("a successful picker setModel must keep the session registered, got: %v", getErr)
		}

		if _, ok := f.settings[sid]; !ok {
			t.Fatal("a successful picker setModel must keep the confirmed settings recorded")
		}

		if _, ok := f.cwds[sid]; !ok {
			t.Fatal("a successful picker setModel must keep the session cwd recorded")
		}

		// No teardown on the wire either: exactly zero session/close frames
		// (the create+subscribe+setModel legs are the only traffic).
		tr := closeLegs()
		for _, line := range tr.Lines {
			if strings.Contains(line.JSON, `"method":"session/close"`) {
				t.Fatalf("a successful switch must not tear the session down — back leg records: %s", line.JSON)
			}
		}
	})
}
