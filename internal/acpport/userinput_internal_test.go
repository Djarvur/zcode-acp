package acpport

import (
	"encoding/json"
	"testing"

	"github.com/coder/acp-go-sdk"
)

// The shared catalog literals (goconst): the ask shapes the build/map
// locks replay.
const (
	elicitSession      = "sess-1"
	elicitPickOne      = "Pick one"
	elicitPhaseOne     = "verify phase one"
	elicitPhaseQ       = "Which phase?"
	elicitVerify       = "Verify 1"
	elicitExit         = "Exit manager"
	elicitExtrasQ      = "Which extras?"
	elicitPlanApproval = "plan_approval"
	elicitApproveLabel = "Approve"
	elicitApproveValue = "approve"
	elicitPlanQ        = "Exit plan mode?"
	elicitPlanPrompt   = "Approve this plan?"
	elicitPlanDesc     = "Exit plan mode and start implementation."
	elicitAccept       = "accept"
)

// elicitOrdinaryParams is the ordinary AskUserQuestion wire body: no
// schema discriminator, and the two label-identified options the ordinary
// arm replays (const = title = label, the answer identity is the label).
const elicitOrdinaryParams = `{
		"requestId": "req-1", "sessionId": "` + elicitSession + `", "toolName": "AskUserQuestion",
		"prompt": "` + elicitPickOne + `",
		"questions": [{"question": "` + elicitPhaseQ + `", "header": "Phase",
			"options": [
				{"value": "` + elicitVerify + `", "label": "` + elicitVerify + `",
					"description": "` + elicitPhaseOne + `"},
				{"value": "` + elicitExit + `", "label": "` + elicitExit + `"}
			],
			"multiSelect": false}]
	}`

// elicitPlanParams is the ExitPlanMode plan-approval wire body: the schema
// discriminator plus the value/title split option.
const elicitPlanParams = `{
		"requestId": "req-plan", "sessionId": "` + elicitSession + `", "toolName": "ExitPlanMode",
		"prompt": "` + elicitPlanPrompt + `",
		"schema": {"interaction": "` + elicitPlanApproval + `", "toolName": "ExitPlanMode"},
		"questions": [{"question": "` + elicitPlanQ + `", "header": "",
			"options": [{"value": "` + elicitApproveValue + `", "label": "` + elicitApproveLabel + `",
				"description": "` + elicitPlanDesc + `"}],
			"multiSelect": false}]
	}`

// planApprovalAsk is the neutral plan-approval ask the build/map locks
// share: the schema discriminator plus the one value/title split option.
func planApprovalAsk() UserInputAsk {
	return UserInputAsk{
		RequestID:   "req-plan",
		SessionID:   elicitSession,
		ToolName:    "ExitPlanMode",
		Prompt:      elicitPlanPrompt,
		Interaction: elicitPlanApproval,
		Questions: []UserInputQuestion{{
			Question: elicitPlanQ,
			Options:  []UserInputOption{planApprovalOption()},
		}},
	}
}

// planApprovalOption is the ExitPlanMode option literal: the wire value is
// the answer identity and the label is the display title.
func planApprovalOption() UserInputOption {
	return UserInputOption{Value: elicitApproveValue, Label: elicitApproveLabel, Description: elicitPlanDesc}
}

// requireStringProperty returns the named string property of the form or
// fails the test (the single-select and free-text arms both use it).
func requireStringProperty(t *testing.T, req acp.CreateElicitationRequest,
) *acp.ElicitationPropertySchemaString {
	t.Helper()

	if req.Form == nil {
		t.Fatal("the elicitation carries no form")
	}

	prop, ok := req.Form.RequestedSchema.Properties["q1"]
	if !ok || prop.String == nil {
		t.Fatalf("q1 property missing or not a string variant: %+v", prop)
	}

	return prop.String
}

// requirePlanApprovalOption pulls the single plan-approval enum option out
// of the q1 property, checking the question title on the way.
func requirePlanApprovalOption(t *testing.T, req acp.CreateElicitationRequest) acp.EnumOption {
	t.Helper()

	prop := requireStringProperty(t, req)
	if prop.Title == nil || *prop.Title != elicitPlanQ {
		t.Fatalf("q1 title = %+v, want the question text", prop.Title)
	}

	if len(prop.OneOf) != 1 {
		t.Fatalf("q1 oneOf = %+v, want the one plan-approval option", prop.OneOf)
	}

	return prop.OneOf[0]
}

// assertPlanApprovalEnumOption locks const = wire value, title = display
// label, and the helper description.
func assertPlanApprovalEnumOption(t *testing.T, opt acp.EnumOption) {
	t.Helper()

	if opt.Const != elicitApproveValue || opt.Title != elicitApproveLabel {
		t.Fatalf("option = %+v, want const=%q title=%q", opt, elicitApproveValue, elicitApproveLabel)
	}

	if opt.Description == nil || *opt.Description != elicitPlanDesc {
		t.Fatalf("option description = %+v, want %q", opt.Description, elicitPlanDesc)
	}
}

// TestBuildElicitationRequest locks the form mapping (UIB-02): one
// index-keyed property per question — single-select → string with titled
// oneOf options (const=label, title=label, description), multiSelect →
// array with titled anyOf items; the message rides the ask's prompt; the
// session scope carries the id.
func TestBuildElicitationRequest(t *testing.T) {
	ask := UserInputAsk{
		RequestID: "req-1",
		SessionID: elicitSession,
		Prompt:    elicitPickOne,
		Questions: []UserInputQuestion{
			{
				Question: elicitPhaseQ,
				Header:   "Phase",
				Options: []UserInputOption{
					{Label: elicitVerify, Description: elicitPhaseOne},
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

// TestBuildElicitationRequestPlanApproval locks the plan-approval arm: the
// enum const is the wire option value, the title stays the display label,
// and an empty wire value falls back to the label identity.
func TestBuildElicitationRequestPlanApproval(t *testing.T) {
	assertPlanApprovalEnumOption(t, requirePlanApprovalOption(t, BuildElicitationRequest(planApprovalAsk())))

	t.Run("empty wire value falls back to the label", func(t *testing.T) {
		ask := planApprovalAsk()
		ask.Questions[0].Options[0].Value = ""

		opt := requirePlanApprovalOption(t, BuildElicitationRequest(ask))
		if opt.Const != elicitApproveLabel || opt.Title != elicitApproveLabel {
			t.Fatalf("option = %+v, want const=title=%q", opt, elicitApproveLabel)
		}
	})
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
		Action: elicitAccept,
		Content: map[string]acp.ElicitationContentValue{
			"q1": acp.NewElicitationContentValueString(elicitExit),
			"q2": acp.NewElicitationContentValueStringArray("lint", "docs"),
		},
	}}

	out := MapElicitationOutcome(accept, ask)
	if out.Action != elicitAccept {
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

// TestMapElicitationOutcomePlanApproval locks the inverse mapping: both the
// enum const and the display title normalize to the wire value; anything
// else passes through verbatim (the backend's allow-only-"approve" gate).
func TestMapElicitationOutcomePlanApproval(t *testing.T) {
	tests := []struct {
		name     string
		selected string
		want     string
	}{
		{name: "the display title", selected: elicitApproveLabel, want: elicitApproveValue},
		{name: "the wire value", selected: elicitApproveValue, want: elicitApproveValue},
		{name: "an unmatched answer", selected: "deny", want: "deny"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := map[string]acp.ElicitationContentValue{
				"q1": acp.NewElicitationContentValueString(tc.selected),
			}
			resp := acp.CreateElicitationResponse{Accept: &acp.CreateElicitationAccept{
				Action:  elicitAccept,
				Content: content,
			}}

			out := MapElicitationOutcome(resp, planApprovalAsk())
			if out.Action != elicitAccept {
				t.Fatalf("action = %q, want accept", out.Action)
			}

			if got := out.Answers[elicitPlanQ]; got != tc.want {
				t.Fatalf("answers[%s] = %q, want %q", elicitPlanQ, got, tc.want)
			}
		})
	}
}

// TestDecodeUserInputAsk locks the neutral ask decode (UIB-01): the full
// AskUserQuestion shape (option values/header/labels/descriptions/
// multiSelect) carries no interaction discriminator, the plan-approval body
// carries its schema discriminator and the value/title split option, the
// no-questions free-text arm stays, and the broken body fails loud.
func TestDecodeUserInputAsk(t *testing.T) {
	assertOrdinaryDecode(t)
	assertPlanApprovalDecode(t)
	assertFreeTextDecode(t)
}

// assertOrdinaryDecode locks the label-identified ordinary params body: no
// interaction discriminator and one populated wire value per option.
func assertOrdinaryDecode(t *testing.T) {
	t.Helper()

	ask, err := DecodeUserInputAsk(json.RawMessage(elicitOrdinaryParams))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if ask.Prompt != elicitPickOne || len(ask.Questions) != 1 {
		t.Fatalf("ask = %+v, want the prompt and one question", ask)
	}

	if ask.Interaction != "" {
		t.Fatalf("ordinary ask interaction = %q, want empty (no schema discriminator)", ask.Interaction)
	}

	if q := ask.Questions[0]; q.Question != elicitPhaseQ || q.Header != "Phase" ||
		len(q.Options) != 2 || q.Options[0].Value != elicitVerify || q.Options[0].Label != elicitVerify ||
		q.Options[0].Description != elicitPhaseOne || q.Options[1].Value != elicitExit {
		t.Fatalf("question = %+v, want the option values, labels, and descriptions", q)
	}
}

// assertPlanApprovalDecode locks the plan-approval params body: the schema
// discriminator and the value/title split option survive the decode.
func assertPlanApprovalDecode(t *testing.T) {
	t.Helper()

	plan, err := DecodeUserInputAsk(json.RawMessage(elicitPlanParams))
	if err != nil {
		t.Fatalf("decode plan approval: %v", err)
	}

	if plan.Interaction != elicitPlanApproval {
		t.Fatalf("plan approval interaction = %q, want %q", plan.Interaction, elicitPlanApproval)
	}

	if len(plan.Questions) != 1 || len(plan.Questions[0].Options) != 1 {
		t.Fatalf("plan approval ask = %+v, want one question with one option", plan)
	}

	if opt := plan.Questions[0].Options[0]; opt.Value != elicitApproveValue ||
		opt.Label != elicitApproveLabel || opt.Description != elicitPlanDesc {
		t.Fatalf("plan approval option = %+v, want the wire value, label, and description", opt)
	}
}

// assertFreeTextDecode locks the no-questions free-text arm and the
// broken-body loud error.
func assertFreeTextDecode(t *testing.T) {
	t.Helper()

	free, err := DecodeUserInputAsk(json.RawMessage(`{"requestId": "r", "prompt": "type"}`))
	if err != nil || len(free.Questions) != 0 || free.Prompt != "type" {
		t.Fatalf("the free-text arm = %+v (err %v), want no questions and the prompt", free, err)
	}

	if _, err := DecodeUserInputAsk(json.RawMessage(`{broken`)); err == nil {
		t.Fatal("a broken body decoded without an error — must fail loud")
	}
}
