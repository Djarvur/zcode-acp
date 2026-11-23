package acpport

import (
	"encoding/json"
	"testing"

	"github.com/coder/acp-go-sdk"
)

// The shared catalog literals (goconst): the ask shapes the build/map
// locks replay.
const (
	elicitSession = "sess-1"
	elicitPhaseQ  = "Which phase?"
	elicitVerify  = "Verify 1"
	elicitExit    = "Exit manager"
	elicitExtrasQ = "Which extras?"
)

// TestBuildElicitationRequest locks the form mapping (UIB-02): one
// index-keyed property per question — single-select → string with titled
// oneOf options (const=label, title=label, description), multiSelect →
// array with titled anyOf items; the message rides the ask's prompt; the
// session scope carries the id.
func TestBuildElicitationRequest(t *testing.T) {
	ask := UserInputAsk{
		RequestID: "req-1",
		SessionID: elicitSession,
		Prompt:    "Pick one",
		Questions: []UserInputQuestion{
			{
				Question: elicitPhaseQ,
				Header:   "Phase",
				Options: []UserInputOption{
					{Label: elicitVerify, Description: "verify phase one"},
					{Label: elicitExit},
				},
			},
			{
				Question:    elicitExtrasQ,
				Header:      "Extras",
				MultiSelect: true,
				Options: []UserInputOption{
					{Label: "lint"},
					{Label: "docs"},
				},
			},
		},
	}

	req := BuildElicitationRequest(ask)

	if req.Form == nil {
		t.Fatal("the elicitation carries no form")
	}

	if req.Form.Message != "Pick one" || req.Form.Mode != "form" {
		t.Fatalf("form message/mode = %q/%q, want the prompt and the form mode", req.Form.Message, req.Form.Mode)
	}

	if req.Form.Session == nil || req.Form.Session.SessionId != elicitSession {
		t.Fatalf("form session scope = %+v, want sess-1", req.Form.Session)
	}

	props := req.Form.RequestedSchema.Properties
	if len(props) != 2 {
		t.Fatalf("schema properties = %d, want 2 (one per question)", len(props))
	}

	assertSingleSelect(t, props)
	assertMultiSelect(t, props)
}

// assertSingleSelect locks the q1 property: the string variant with the
// question title and the two titled oneOf options.
func assertSingleSelect(t *testing.T, props map[string]acp.ElicitationPropertySchema) {
	t.Helper()

	single, ok := props["q1"]
	if !ok || single.String == nil {
		t.Fatalf("q1 property missing or not a string variant: %+v", single)
	}

	if single.String.Title == nil || *single.String.Title != elicitPhaseQ {
		t.Fatalf("q1 title = %+v, want the question text", single.String.Title)
	}

	if len(single.String.OneOf) != 2 {
		t.Fatalf("q1 oneOf = %+v, want the two titled options", single.String.OneOf)
	}

	if single.String.OneOf[0].Const != elicitVerify || single.String.OneOf[0].Title != elicitVerify {
		t.Fatalf("q1 option[0] = %+v, want const=title=Verify 1", single.String.OneOf[0])
	}

	if single.String.OneOf[0].Description == nil || *single.String.OneOf[0].Description != "verify phase one" {
		t.Fatalf("q1 option[0] description = %+v, want the option description", single.String.OneOf[0].Description)
	}
}

// assertMultiSelect locks the q2 property: the titled array variant with
// the two label options.
func assertMultiSelect(t *testing.T, props map[string]acp.ElicitationPropertySchema) {
	t.Helper()

	multi, ok := props["q2"]
	if !ok || multi.Array == nil || multi.Array.Items.Titled == nil {
		t.Fatalf("q2 property missing or not a titled array variant: %+v", multi)
	}

	if len(multi.Array.Items.Titled.AnyOf) != 2 {
		t.Fatalf("q2 anyOf = %+v, want the two titled options", multi.Array.Items.Titled.AnyOf)
	}
}

// TestBuildElicitationRequestFreeText — the no-questions arm: the form
// carries ONE text property keyed off the prompt (the free-text answer
// maps to content.answer per the broker's compat path).
func TestBuildElicitationRequestFreeText(t *testing.T) {
	req := BuildElicitationRequest(UserInputAsk{
		SessionID: elicitSession,
		Prompt:    "type something",
	})

	if req.Form == nil || len(req.Form.RequestedSchema.Properties) != 1 {
		t.Fatalf("free-text form = %+v, want one prompt property", req.Form)
	}
}

// TestMapElicitationOutcome locks the answer mapping (UIB-03): accept
// content keys (q1..qn) map back to the QUESTION TEXT with the option
// labels (multi joined ", "); decline/cancel map verbatim.
func TestMapElicitationOutcome(t *testing.T) {
	ask := UserInputAsk{
		Questions: []UserInputQuestion{
			{Question: elicitPhaseQ, Options: []UserInputOption{{Label: "Verify 1"}, {Label: "Exit manager"}}},
			{Question: elicitExtrasQ, MultiSelect: true, Options: []UserInputOption{{Label: "lint"}, {Label: "docs"}}},
		},
	}

	accept := acp.CreateElicitationResponse{Accept: &acp.CreateElicitationAccept{
		Action: "accept",
		Content: map[string]acp.ElicitationContentValue{
			"q1": acp.NewElicitationContentValueString(elicitExit),
			"q2": acp.NewElicitationContentValueStringArray("lint", "docs"),
		},
	}}

	out := MapElicitationOutcome(accept, ask)
	if out.Action != "accept" {
		t.Fatalf("action = %q, want accept", out.Action)
	}

	if out.Answers[elicitPhaseQ] != elicitExit {
		t.Fatalf("answers[Which phase?] = %q, want the selected label", out.Answers["Which phase?"])
	}

	if out.Answers[elicitExtrasQ] != "lint, docs" {
		t.Fatalf("answers[Which extras?] = %q, want the multi-select joined", out.Answers["Which extras?"])
	}

	decline := acp.CreateElicitationResponse{Decline: &acp.CreateElicitationDecline{Action: "decline"}}
	if out := MapElicitationOutcome(decline, ask); out.Action != "decline" {
		t.Fatalf("decline mapped to %q", out.Action)
	}

	cancel := acp.CreateElicitationResponse{Cancel: &acp.CreateElicitationCancel{Action: "cancel"}}
	if out := MapElicitationOutcome(cancel, ask); out.Action != "cancel" {
		t.Fatalf("cancel mapped to %q", out.Action)
	}
}

// TestDecodeUserInputAsk locks the neutral ask decode (UIB-01): the full
// AskUserQuestion shape (header/labels/descriptions/multiSelect), the
// no-questions free-text arm, and the broken-body loud error.
func TestDecodeUserInputAsk(t *testing.T) {
	full := json.RawMessage(`{
		"requestId": "req-1", "sessionId": "sess-1", "toolName": "AskUserQuestion",
		"prompt": "Pick one",
		"questions": [{"question": "Which phase?", "header": "Phase",
			"options": [{"value": "Verify 1", "label": "Verify 1", "description": "verify phase one"}],
			"multiSelect": false}]
	}`)

	ask, err := DecodeUserInputAsk(full)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if ask.Prompt != "Pick one" || len(ask.Questions) != 1 {
		t.Fatalf("ask = %+v, want the prompt and one question", ask)
	}

	if q := ask.Questions[0]; q.Question != "Which phase?" || q.Header != "Phase" ||
		len(q.Options) != 1 || q.Options[0].Label != "Verify 1" || q.Options[0].Description != "verify phase one" {
		t.Fatalf("question = %+v, want the option labels and descriptions", q)
	}

	free, err := DecodeUserInputAsk(json.RawMessage(`{"requestId": "r", "prompt": "type"}`))
	if err != nil || len(free.Questions) != 0 || free.Prompt != "type" {
		t.Fatalf("the free-text arm = %+v (err %v), want no questions and the prompt", free, err)
	}

	if _, err := DecodeUserInputAsk(json.RawMessage(`{broken`)); err == nil {
		t.Fatal("a broken body decoded without an error — must fail loud")
	}
}
