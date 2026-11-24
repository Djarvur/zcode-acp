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
//	TestPickerSetModelRefusalRunsFullTeardown — a fatal/dead-target backend
//	  setModel on an ESTABLISHED session must run the full SESS-01 teardown: the
//	  registry holds no entry for the id and the cwd/settings maps are
//	  empty (the zombie gone — the RED arm), the client still receives the
//	  coded setModel refusal (the reply is preserved), a follow-up prompt
//	  answers the closing/closed invalid-request class and never reaches
//	  the backend wire, and a SUCCESSFUL switch keeps the session fully
//	  alive (the teardown fires only on the refusal arm).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	// pickerFlashModelID is the reply-confirmed retry target.
	pickerFlashModelID = "GLM-5.3-Flash"

	// pickerFlashValue is the picker target (the providerId/modelId
	// serialization the snapshot's model option lists).
	pickerFlashValue = "zai-api/" + pickerFlashModelID

	// pickerValidationMessage is the confirmed field-refusal shape used by
	// the structured-data precedence and lifecycle locks.
	pickerValidationMessage = `Reasoning effort "max" is not supported by OpenCodeGo/hungerg-free`
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

// pickerFlashSnapshot is a minimal successful setModel reply confirming the
// picker target; recordMergedSettings retains the previous model catalog.
func pickerFlashSnapshot() json.RawMessage {
	return json.RawMessage(`{
	  "session": {"sessionId": "` + pickerSID + `"},
	  "settings": {"model": {"current": {
	    "providerId": "zai-api", "modelId": "GLM-5.3-Flash"
	  }}}
	}`)
}

// pickerRefusalScenario scripts the established session's back leg: the
// create + subscribe preamble answers, the picker's setModel carries the
// refused arm, the teardown's session/close answers (the full SESS-01
// teardown issues one), and any session/send fails loud (a send after the
// refusal is exactly the zombie arm this lock forbids).
func pickerRefusalScenario(setModelRules ...fake.RequestRule) *fake.Scenario {
	rules := make([]fake.RequestRule, 0, 2+len(setModelRules)+1)
	rules = append(rules,
		fake.RequestRule{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: pickerSnapshot()}},
		fake.RequestRule{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{
			Result: json.RawMessage(`{"sessionId":"` + pickerSID + `","eventSeq":0,"events":[]}`),
		}},
	)
	rules = append(rules, setModelRules...)
	rules = append(rules, fake.RequestRule{
		Method: zcode.MethodSessionClose,
		Reply:  &fake.Reply{Result: json.RawMessage(`{"closed":true}`)},
	})

	return &fake.Scenario{
		Name:      "picker-refusal-teardown",
		OnRequest: rules,
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

// assertPickerSessionPreserved locks the recoverable-refusal contract: the
// live registry entry, cwd, and byte-identical confirmed settings all survive.
func assertPickerSessionPreserved(t *testing.T, f *Front, sid, cwd string, before []byte) {
	t.Helper()

	if _, getErr := f.registry.Get(sid); getErr != nil {
		t.Fatalf("a recoverable setModel refusal dropped the registered session: %v", getErr)
	}

	if got := f.sessionCwdLocked(sid); got != cwd {
		t.Fatalf("a recoverable setModel refusal changed cwd to %q, want %q", got, cwd)
	}

	after, ok := f.latestSettings(sid)
	if !ok {
		t.Fatal("a recoverable setModel refusal dropped the confirmed settings")
	}

	if !bytes.Equal(after, before) {
		t.Fatalf("a recoverable setModel refusal changed confirmed settings:\n got: %s\nwant: %s", after, before)
	}
}

// assertPickerCloseCount counts the backend session/close frames on the
// composed leg — zero for a preserved session, one for fatal teardown.
func assertPickerCloseCount(t *testing.T, tr fake.Transcript, want int) {
	t.Helper()

	closes := 0
	for _, line := range tr.Lines {
		if string(line.Dir) == "in" && strings.Contains(line.JSON, `"method":"session/close"`) {
			closes++
		}
	}

	if closes != want {
		t.Fatalf("back leg records %d session/close frames, want exactly %d", closes, want)
	}
}

// assertPickerCurrentFlash requires the stored snapshot to confirm Flash as
// the current model after a successful retry.
func assertPickerCurrentFlash(t *testing.T, f *Front, sid string) json.RawMessage {
	t.Helper()

	updated, ok := f.latestSettings(sid)
	if !ok {
		t.Fatal("the successful retry must record confirmed settings")
	}

	s, parsed := ParseSettings(updated)
	if !parsed || s.Model.Current == nil || s.Model.Current.ModelID != pickerFlashModelID {
		t.Fatalf("the successful retry did not record Flash as current: %s", updated)
	}

	return updated
}

// assertPickerSetModelCount requires exactly want backend setModel frames.
func assertPickerSetModelCount(t *testing.T, tr fake.Transcript, want int) {
	t.Helper()

	setModels := 0
	for _, line := range tr.Lines {
		if string(line.Dir) == "in" && strings.Contains(line.JSON, `"method":"session/setModel"`) {
			setModels++
		}
	}

	if setModels != want {
		t.Fatalf("back leg records %d session/setModel frames, want exactly %d", setModels, want)
	}
}

// TestPickerRecoverableSetModelValidationPreservesSession proves the real
// picker path preserves a live session after a recoverable selection refusal,
// then records the confirmed settings from a later valid reply. Both the
// structured backend error and the data-less field-message compatibility arm
// ride the composed fake leg and the exported picker handler.
func TestPickerRecoverableSetModelValidationPreservesSession(t *testing.T) {
	const message = pickerValidationMessage

	for _, tc := range []struct {
		name string
		data json.RawMessage
	}{
		{
			name: "structured ModelProtocolError",
			data: json.RawMessage(`{"name":"ModelProtocolError","code":"invalid_model_request"}`),
		},
		{name: "data-less field-message fallback"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			ctx := context.Background()

			refusal := &zcode.WireError{Code: zcode.CodeInternalError, Message: message, Data: tc.data}
			success := fake.RequestRule{
				Method: zcode.MethodSessionSetModel,
				Reply:  &fake.Reply{Result: pickerFlashSnapshot()},
			}
			f, closeLegs := newInterleaveFront(t, pickerRefusalScenario(
				fake.RequestRule{Method: zcode.MethodSessionSetModel, Times: 1, Reply: &fake.Reply{Error: refusal}},
				success,
			))
			defer closeLegs()

			sid := openPickerSession(ctx, t, f, cwd)
			before, ok := f.latestSettings(sid)
			if !ok {
				t.Fatal("the create reply must record the initial confirmed settings")
			}

			_, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerFlashValue)
			var seamErr *acpport.Error
			if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInternal {
				t.Fatalf("refusal error = %v, want the coded internal setModel refusal", err)
			}

			wantMessage := "session/setModel: zcode: wire error -32603: " + message
			if seamErr.Message != wantMessage {
				t.Fatalf("refusal message = %q, want unchanged %q", seamErr.Message, wantMessage)
			}

			assertPickerSessionPreserved(t, f, sid, cwd, before)

			if _, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerFlashValue); err != nil {
				t.Fatalf("the later valid picker switch must succeed on the preserved session: %v", err)
			}

			updated := assertPickerCurrentFlash(t, f, sid)

			assertPickerSessionPreserved(t, f, sid, cwd, updated)
			assertPickerCloseCount(t, closeLegs(), 0)
		})
	}
}

// TestIsRecoverableSetModelError locks the conservative trust-boundary
// classifier, especially that present malformed/unrecognized -32603 data
// can never fall through to the validation-shaped message compatibility arm.
//
//nolint:funlen // the classification matrix is one trust-boundary contract
func TestIsRecoverableSetModelError(t *testing.T) {
	wire := func(code int, message string, data json.RawMessage) error {
		return &zcode.CallError{Err: &zcode.WireError{Code: code, Message: message, Data: data}}
	}

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "wrapped invalid params ignores unrelated message and malformed data",
			err:  fmt.Errorf("wrapped: %w", wire(zcode.CodeInvalidParams, "unrelated", json.RawMessage(`{`))),
			want: true,
		},
		{
			name: "structured invalid model request ignores non-validation message",
			err: wire(zcode.CodeInternalError, "request refused", json.RawMessage(
				`{"name":"ModelProtocolError","code":"invalid_model_request"}`)),
			want: true,
		},
		{
			name: "data-less unsupported effort",
			err:  wire(zcode.CodeInternalError, pickerValidationMessage, nil),
			want: true,
		},
		{
			name: "data-less unsupported level",
			err:  wire(zcode.CodeInternalError, `Reasoning level "max" is not supported by a model`, nil),
			want: true,
		},
		{name: "nil", err: nil},
		{name: "non-call error", err: fmt.Errorf("transport failed: %w", zcode.ErrConnClosed)},
		{name: "call error with nil wire error", err: &zcode.CallError{}},
		{name: "typed nil call error", err: (*zcode.CallError)(nil)},
		{
			name: "resource not found",
			err:  wire(zcode.CodeResourceNotFound, pickerValidationMessage, nil),
		},
		{
			name: "method not found",
			err:  wire(zcode.CodeMethodNotFound, pickerValidationMessage, nil),
		},
		{
			name: "generic internal error",
			err:  wire(zcode.CodeInternalError, "unknown backend failure", nil),
		},
		{
			name: "old reasoning-required message",
			err:  wire(zcode.CodeInternalError, "Reasoning level is required for this model", nil),
		},
		{name: "connection closed", err: zcode.ErrConnClosed},
		{name: "cancellation", err: context.Canceled},
		{
			name: "present invalid JSON never reaches message fallback",
			err:  wire(zcode.CodeInternalError, pickerValidationMessage, json.RawMessage(`{`)),
		},
		{
			name: "present JSON null never reaches message fallback",
			err:  wire(zcode.CodeInternalError, pickerValidationMessage, json.RawMessage(`null`)),
		},
		{
			name: "present empty object never reaches message fallback",
			err:  wire(zcode.CodeInternalError, pickerValidationMessage, json.RawMessage(`{}`)),
		},
		{
			name: "wrong code field type never reaches message fallback",
			err: wire(zcode.CodeInternalError, pickerValidationMessage,
				json.RawMessage(`{"name":"ModelProtocolError","code":17}`)),
		},
		{
			name: "unknown name and code never reach message fallback",
			err: wire(zcode.CodeInternalError, pickerValidationMessage,
				json.RawMessage(`{"name":"OtherError","code":"other_code"}`)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRecoverableSetModelError(tc.err); got != tc.want {
				t.Fatalf("isRecoverableSetModelError(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

// TestPickerMalformedOrUnrecognizedSetModelDataRunsFullTeardown proves the
// same present-data boundary through the real picker lifecycle: a validation-
// shaped message cannot rescue malformed or unrecognized structured data.
func TestPickerMalformedOrUnrecognizedSetModelDataRunsFullTeardown(t *testing.T) {
	for _, tc := range []struct {
		name string
		data json.RawMessage
	}{
		{name: "JSON null", data: json.RawMessage(`null`)},
		{
			name: "wrong code field type",
			data: json.RawMessage(`{"name":"ModelProtocolError","code":17}`),
		},
		{
			name: "unknown name and code",
			data: json.RawMessage(`{"name":"OtherError","code":"other_code"}`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			ctx := context.Background()
			f, closeLegs := newInterleaveFront(t, pickerRefusalScenario(fake.RequestRule{
				Method: zcode.MethodSessionSetModel,
				Reply: &fake.Reply{Error: &zcode.WireError{
					Code: zcode.CodeInternalError, Message: pickerValidationMessage, Data: tc.data,
				}},
			}))
			defer closeLegs()

			sid := openPickerSession(ctx, t, f, cwd)

			_, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerFlashValue)
			var seamErr *acpport.Error
			if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInternal {
				t.Fatalf("refusal error = %v, want the coded internal setModel refusal", err)
			}

			assertPickerZombieGone(t, f, "malformed or unrecognized structured setModel data")
			assertPickerCloseCount(t, closeLegs(), 1)
		})
	}
}

// TestModelSlashRecoverableSetModelValidationPreservesSession drives the
// direct /model intercept: recoverable validation preserves the live session,
// and a later valid command succeeds with reply-confirmed settings and no
// session/close frame.
func TestModelSlashRecoverableSetModelValidationPreservesSession(t *testing.T) {
	cwd := t.TempDir()
	ctx := context.Background()
	f, closeLegs := newInterleaveFront(t, pickerRefusalScenario(
		fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Times:  1,
			Reply: &fake.Reply{Error: &zcode.WireError{
				Code: zcode.CodeInternalError, Message: pickerValidationMessage,
			}},
		},
		fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: pickerFlashSnapshot()},
		},
	))
	defer closeLegs()

	sid := openPickerSession(ctx, t, f, cwd)
	before, ok := f.latestSettings(sid)
	if !ok {
		t.Fatal("the create reply must record the initial confirmed settings")
	}

	ask := acpport.PromptParams{SessionID: sid, Text: modelSlashSpace + pickerFlashValue}
	_, handled, err := f.promptModelSlash(ctx, ask, pickerNoopSink())
	if !handled {
		t.Fatal("the /model selector must be handled by the slash intercept")
	}

	var seamErr *acpport.Error
	if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInternal {
		t.Fatalf("refusal error = %v, want the coded internal setModel refusal", err)
	}

	wantMessage := "session/setModel: zcode: wire error -32603: " + pickerValidationMessage
	if seamErr.Message != wantMessage {
		t.Fatalf("refusal message = %q, want unchanged %q", seamErr.Message, wantMessage)
	}

	assertPickerSessionPreserved(t, f, sid, cwd, before)

	stop, handled, err := f.promptModelSlash(ctx, ask, pickerNoopSink())
	if !handled || err != nil || stop != acpport.StopEndTurn {
		t.Fatalf("the later valid /model switch = (%q, %t, %v), want (end_turn, true, nil)",
			stop, handled, err)
	}

	updated := assertPickerCurrentFlash(t, f, sid)

	assertPickerSessionPreserved(t, f, sid, cwd, updated)

	tr := closeLegs()
	assertPickerSetModelCount(t, tr, 2)
	assertPickerCloseCount(t, tr, 0)
}

// TestPickerSetModelRefusalRunsFullTeardown locks the WR-01 contract: a
// fatal/dead-target picker setModel fully tears the front session down (no
// zombie), the refusal reply is preserved, the follow-up prompt lands the coded
// closing/closed class without a backend send, and the successful switch
// keeps the session fully alive.
//
//nolint:funlen,contextcheck // full harness locks; the harness owns its conn lifecycle (NewConn is ctx-free)
func TestPickerSetModelRefusalRunsFullTeardown(t *testing.T) {
	cwd := t.TempDir()
	ctx := context.Background()

	refusalRule := fake.RequestRule{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{
		Error: &zcode.WireError{Code: zcode.CodeResourceNotFound, Message: "session not found"},
	}}

	t.Run("the fatal refusal tears the front session down (no zombie)", func(t *testing.T) {
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

const (
	pickerDeepSeekValue    = "zai-api/DeepSeek"
	pickerHungergFreeValue = "OpenCodeGo/hungerg-free"
)

// pickerCrossModelCreateSnapshot models the state already preserved after the
// historical recoverable refusal: DeepSeek/max is confirmed, while the exact
// retry target advertises its own low/high levels and high default.
func pickerCrossModelCreateSnapshot() json.RawMessage {
	return json.RawMessage(`{
	  "session": {"sessionId": "` + pickerSID + `"},
	  "settings": {
	    "model": {
	      "current": {"providerId": "zai-api", "modelId": "DeepSeek"},
	      "available": [
	        {"ref": {"providerId": "zai-api", "modelId": "DeepSeek"}, "label": "DeepSeek",
	         "reasoning": {"levels": [
	            {"value": "low", "label": "Low"},
	            {"value": "high", "label": "High"},
	            {"value": "max", "label": "Max"}], "defaultLevel": "max"}},
	        {"ref": {"providerId": "OpenCodeGo", "modelId": "hungerg-free"}, "label": "hungerg-free",
	         "reasoning": {"levels": [
	            {"value": "low", "label": "Low"},
	            {"value": "high", "label": "High"}], "defaultLevel": "high"}}
	      ]
	    },
	    "thoughtLevel": {
	      "enabled": true, "current": "max", "defaultLevel": "max",
	      "available": [{"value": "max", "label": "Max"}]
	    }
	  }
	}`)
}

// pickerCrossModelReply confirms the one post-fix retry: the target is current
// with its advertised high level. The prior snapshot remains the catalog source.
func pickerCrossModelReply() json.RawMessage {
	return json.RawMessage(`{
	  "session": {"sessionId": "` + pickerSID + `"},
	  "settings": {
	    "model": {
	      "current": {"providerId": "OpenCodeGo", "modelId": "hungerg-free"}
	    },
	    "thoughtLevel": {
	      "enabled": true, "current": "high", "defaultLevel": "high",
	      "available": [{"value": "high", "label": "High"}]
	    }
	  }
	}`)
}

// assertPickerReasoningState requires the stored backend-confirmed snapshot to
// name the exact model and reasoning level.
func assertPickerReasoningState(t *testing.T, f *Front, sid, providerID, modelID, level string) {
	t.Helper()

	raw, ok := f.latestSettings(sid)
	if !ok {
		t.Fatal("the confirmed settings snapshot must remain recorded")
	}

	s, parsed := ParseSettings(raw)
	if !parsed || s.Model.Current == nil {
		t.Fatalf("the confirmed settings must carry a current model: %s", raw)
	}

	if s.Model.Current.ProviderID != providerID || s.Model.Current.ModelID != modelID {
		t.Fatalf("current model = %s/%s, want %s/%s: %s",
			s.Model.Current.ProviderID, s.Model.Current.ModelID, providerID, modelID, raw)
	}

	if got := confirmedLevel(s); got != level {
		t.Fatalf("confirmed reasoning level = %q, want %q: %s", got, level, raw)
	}
}

// assertPickerSetModelPayload decodes the sole inbound setModel frame and
// locks the exact target, target-supported level, and explicit persist-false.
func assertPickerSetModelPayload(t *testing.T, tr fake.Transcript, providerID, modelID, level string) {
	t.Helper()

	assertPickerSetModelCount(t, tr, 1)

	for _, line := range tr.Lines {
		if string(line.Dir) != "in" || !strings.Contains(line.JSON, `"method":"session/setModel"`) {
			continue
		}

		var frame struct {
			Params struct {
				Model struct {
					ProviderID string `json:"providerId"`
					ModelID    string `json:"modelId"`
					Options    *struct {
						ReasoningLevel string `json:"reasoningLevel"`
					} `json:"options"`
				} `json:"model"`
				PersistAsWorkspaceLastUsed *bool `json:"persistAsWorkspaceLastUsed"`
			} `json:"params"`
		}
		if err := json.Unmarshal([]byte(line.JSON), &frame); err != nil {
			t.Fatalf("decode setModel frame: %v (%s)", err, line.JSON)
		}

		model := frame.Params.Model
		if model.ProviderID != providerID || model.ModelID != modelID {
			t.Fatalf("setModel model = %s/%s, want %s/%s: %s",
				model.ProviderID, model.ModelID, providerID, modelID, line.JSON)
		}

		if model.Options == nil || model.Options.ReasoningLevel != level {
			t.Fatalf("setModel options = %+v, want reasoningLevel %q: %s", model.Options, level, line.JSON)
		}

		if frame.Params.PersistAsWorkspaceLastUsed == nil || *frame.Params.PersistAsWorkspaceLastUsed {
			t.Fatalf("setModel persistAsWorkspaceLastUsed must be explicitly false: %s", line.JSON)
		}

		return
	}

	t.Fatal("the sole setModel frame was not found")
}

// TestPickerCrossModelRetryResolvesTargetReasoning starts from the observed
// already-preserved DeepSeek/max snapshot and proves one retry sends the exact
// target's advertised high without driving the historical refusal lifecycle.
//
//nolint:funlen // one composed preservation/retry contract; splitting would hide its causal sequence
func TestPickerCrossModelRetryResolvesTargetReasoning(t *testing.T) {
	cwd := t.TempDir()
	ctx := context.Background()

	f, closeLegs := newInterleaveFront(t, &fake.Scenario{
		Name: "picker-cross-model-post-preservation-retry",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: pickerCrossModelCreateSnapshot()}},
			{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{
				Result: json.RawMessage(`{"sessionId":"` + pickerSID + `","eventSeq":0,"events":[]}`),
			}},
			{Method: zcode.MethodSessionSetModel, Times: 1, Reply: &fake.Reply{Result: pickerCrossModelReply()}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: "MethodNotFound"}},
		EOFDrainMs:    1,
	})
	defer closeLegs()

	sid := openPickerSession(ctx, t, f, cwd)
	beforeRegistry, err := f.registry.Get(sid)
	if err != nil {
		t.Fatalf("capture registry baseline: %v", err)
	}

	before, ok := f.latestSettings(sid)
	if !ok {
		t.Fatal("the create reply must record the already-preserved baseline")
	}

	assertPickerReasoningState(t, f, sid, "zai-api", "DeepSeek", "max")
	assertPickerSessionPreserved(t, f, sid, cwd, before)

	if _, err := f.SetSessionConfigOption(ctx, sid, optionIDModel, pickerHungergFreeValue); err != nil {
		t.Fatalf("the single post-preservation retry must succeed: %v", err)
	}

	assertPickerReasoningState(t, f, sid, "OpenCodeGo", "hungerg-free", "high")

	afterRegistry, err := f.registry.Get(sid)
	if err != nil {
		t.Fatalf("the successful retry dropped registry membership: %v", err)
	}

	if afterRegistry.ID != beforeRegistry.ID || afterRegistry.Machine != beforeRegistry.Machine {
		t.Fatal("the successful retry replaced the registered session machinery")
	}

	if got := f.sessionCwdLocked(sid); got != cwd {
		t.Fatalf("the successful retry changed cwd to %q, want %q", got, cwd)
	}

	tr := closeLegs()
	assertPickerSetModelPayload(t, tr, "OpenCodeGo", "hungerg-free", "high")
	assertPickerCloseCount(t, tr, 0)
}

// TestSetConfigMutationMissingTargetFailsClosed proves a direct mutation cannot
// borrow source/global levels when the requested target has no catalog entry.
func TestSetConfigMutationMissingTargetFailsClosed(t *testing.T) {
	s := Settings{
		Model: ModelSettings{Available: []ModelEntry{{Ref: ModelRef{ProviderID: "zai-api", ModelID: "DeepSeek"}}}},
		ThoughtLevel: ThoughtLevelSettings{
			Enabled: true,
			Current: "max",
		},
	}

	payload, err := setConfigMutation(pickerSID, s, optionIDModel, pickerHungergFreeValue)
	if err == nil {
		t.Fatal("a target absent from settings.model.available must fail closed")
	}

	var seamErr *acpport.Error
	if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInvalidParams {
		t.Fatalf("missing-target error = %v, want coded invalid params", err)
	}

	if payload != nil {
		t.Fatalf("a missing target must return no payload, got %+v", payload)
	}
}
