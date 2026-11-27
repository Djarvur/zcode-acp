package acpport

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/coder/acp-go-sdk"
)

// The ask-the-user bridge's SDK mapping (v0.22.0, UIB-02/UIB-03): the
// neutral ask renders as ONE form elicitation (one index-keyed property per
// question — duplicate question texts can never collide), and the client's
// outcome maps back into the dialect's answers merge contract
// (`content.answers` keyed by QUESTION TEXT —
// `userInputResponseToBrokerResult` + `normalizeAskUserQuestionResponseContent`
// merge it over the tool input; zcode-src interaction-broker.ts).
//
// Two answer identities share this bridge, switched ONLY by the params
// schema.interaction discriminator: ordinary AskUserQuestion asks keep label
// identity end to end (const = title = label; the selected label is the
// answer), while ExitPlanMode's plan approval uses wire-value identity (the
// enum const is the option value, the title is the display label, and either
// accepted string maps back to the value — the backend's
// normalizePlanApprovalAnswer allows only the literal "approve").

// interactionPlanApproval is the params schema.interaction discriminator
// ZCode's ExitPlanMode sends with its plan-approval ask; it is the only
// signal that switches the bridge from label identity to wire-value identity
// (never the tool name).
const interactionPlanApproval = "plan_approval"

// UserInputOption is one choice. Value is the wire option value and the
// answer identity for plan approval; Label is the display title and, for
// ordinary asks, the answer identity (const = title = label). Description is
// the optional helper line.
type UserInputOption struct {
	Value       string
	Label       string
	Description string
}

// UserInputQuestion is one question with its fixed options.
type UserInputQuestion struct {
	Question    string
	Header      string
	Options     []UserInputOption
	MultiSelect bool
}

// UserInputAsk is one decoded ask-the-user request. Interaction is the
// params schema.interaction discriminator (empty for ordinary asks).
type UserInputAsk struct {
	RequestID   string
	SessionID   string
	ToolName    string
	Prompt      string
	Interaction string
	Questions   []UserInputQuestion
}

// UserInputAsker renders one ask on a UI surface and reports the outcome.
// An error means the surface failed — the caller answers decline with the
// reason (never fails the turn).
type UserInputAsker func(ctx context.Context, ask UserInputAsk) (UserInputOutcome, error)

// ElicitationSender is the client-connection seam: the SDK's
// AgentSideConnection.CreateElicitation method value fits it directly.
type ElicitationSender func(ctx context.Context,
	req acp.CreateElicitationRequest,
) (acp.CreateElicitationResponse, error)

// NewUserInputAsker composes the ask bridge over one elicitation sender
// (the D-01 seam: the SDK types stay inside this package — main wires
// `NewUserInputAsker(agentConn.CreateElicitation)` with no SDK reference).
func NewUserInputAsker(send ElicitationSender) UserInputAsker {
	return func(ctx context.Context, ask UserInputAsk) (UserInputOutcome, error) {
		resp, err := send(ctx, BuildElicitationRequest(ask))
		if err != nil {
			return UserInputOutcome{}, err
		}

		return MapElicitationOutcome(resp, ask), nil
	}
}

// UserInputOutcome is the bridged answer: Action accept/decline/cancel and,
// on accept, the answers keyed by QUESTION TEXT with the selected identity
// (the label for ordinary asks, the wire value for plan approval — the
// dialect's content.answers merge contract).
type UserInputOutcome struct {
	Action  string // "accept" | "decline" | "cancel"
	Answers map[string]string
}

// DecodeUserInputAsk digs the ask-the-user shapes out of one
// interaction/requestUserInput params body. The no-questions shape decodes
// to the free-text arm (the prompt only); a broken body fails loud — the
// caller's decline-with-reason needs the distinction. The params
// schema.interaction discriminator and each option's wire value ride the
// neutral ask (plan approval needs both; ordinary asks keep label identity).
func DecodeUserInputAsk(params json.RawMessage) (UserInputAsk, error) {
	var raw struct {
		RequestID string `json:"requestId"` //nolint:tagliatelle // capture-verbatim wire keys (the dialect's camelCase)
		SessionID string `json:"sessionId"` //nolint:tagliatelle // capture-verbatim
		ToolName  string `json:"toolName"`  //nolint:tagliatelle // capture-verbatim
		Prompt    string `json:"prompt"`
		Schema    *struct {
			Interaction string `json:"interaction"`
		} `json:"schema"`
		Questions []struct {
			Question string `json:"question"`
			Header   string `json:"header"`
			Options  []struct {
				Value       string `json:"value"`
				Label       string `json:"label"`
				Description string `json:"description"`
			} `json:"options"`
			MultiSelect bool `json:"multiSelect"` //nolint:tagliatelle // capture-verbatim
		} `json:"questions"`
	}

	if err := json.Unmarshal(params, &raw); err != nil {
		return UserInputAsk{}, fmt.Errorf("interact: requestUserInput params: %w", err)
	}

	ask := UserInputAsk{
		RequestID: raw.RequestID,
		SessionID: raw.SessionID,
		ToolName:  raw.ToolName,
		Prompt:    raw.Prompt,
	}

	if raw.Schema != nil {
		ask.Interaction = raw.Schema.Interaction
	}

	for _, q := range raw.Questions {
		qq := UserInputQuestion{
			Question:    q.Question,
			Header:      q.Header,
			MultiSelect: q.MultiSelect,
		}

		for _, o := range q.Options {
			qq.Options = append(qq.Options, UserInputOption{
				Value:       o.Value,
				Label:       o.Label,
				Description: o.Description,
			})
		}

		ask.Questions = append(ask.Questions, qq)
	}

	return ask, nil
}

// BuildElicitationRequest renders one decoded ask as the ACP form
// elicitation the client's composer dialog takes.
func BuildElicitationRequest(ask UserInputAsk) acp.CreateElicitationRequest {
	props := map[string]acp.ElicitationPropertySchema{}
	message := ask.Prompt

	for i, q := range ask.Questions {
		key := elicitKey(i)

		if message == "" {
			message = q.Question
		}

		opts := make([]acp.EnumOption, 0, len(q.Options))
		for _, o := range q.Options {
			opt := acp.EnumOption{Const: optionIdentity(ask.Interaction, o), Title: o.Label}
			if o.Description != "" {
				d := o.Description
				opt.Description = &d
			}

			opts = append(opts, opt)
		}

		title := q.Question

		if q.MultiSelect {
			props[key] = acp.ElicitationPropertySchema{Array: &acp.ElicitationPropertySchemaArray{
				Items:       acp.MultiSelectItems{Titled: &acp.TitledMultiSelectItems{AnyOf: opts}},
				Description: strPtr(questionLine(q)),
			}}
		} else {
			props[key] = acp.ElicitationPropertySchema{String: &acp.ElicitationPropertySchemaString{
				Title:       &title,
				OneOf:       opts,
				Description: strPtr(questionLine(q)),
			}}
		}
	}

	// The no-questions arm (the broker's free-text path): one text
	// property keyed "answer" — the mapped value lands in the broker's
	// content.answer compat slot.
	if len(ask.Questions) == 0 {
		props["answer"] = acp.ElicitationPropertySchema{String: &acp.ElicitationPropertySchemaString{
			Title: strPtr("Your answer"),
		}}
	}

	if message == "" {
		message = "The agent needs your input"
	}

	return acp.CreateElicitationRequest{Form: &acp.CreateElicitationForm{
		Message: message,
		Mode:    "form",
		RequestedSchema: acp.ElicitationSchema{
			Properties: props,
		},
		Session: &acp.ElicitationSessionScope{SessionId: acp.SessionId(ask.SessionID)},
	}}
}

// MapElicitationOutcome translates the client's elicitation outcome into
// the neutral ask outcome: accept content keys (q1..qn) resolve back to
// the question texts with the selected identity — the label for ordinary
// asks, the wire value for plan approval (multi-select joins ", ") — and
// the free-text arm answers content.answer via the answer map's prompt key
// (the broker's compat slot). decline/cancel pass verbatim.
func MapElicitationOutcome(resp acp.CreateElicitationResponse, ask UserInputAsk) UserInputOutcome {
	switch {
	case resp.Accept != nil:
		answers := map[string]string{}

		for i, q := range ask.Questions {
			v, ok := resp.Accept.Content[elicitKey(i)]
			if !ok {
				continue
			}

			if v.StringArray != nil {
				selected := make([]string, 0, len(*v.StringArray))
				for _, s := range *v.StringArray {
					selected = append(selected, answerSelection(ask.Interaction, q.Options, s))
				}

				answers[q.Question] = strings.Join(selected, ", ")

				continue
			}

			if v.String != nil {
				answers[q.Question] = answerSelection(ask.Interaction, q.Options, string(*v.String))
			}
		}

		// The no-questions (free-text) arm: the deterministic "answer"
		// property lands in the broker's content.answer compat slot.
		if len(ask.Questions) == 0 {
			if v, ok := resp.Accept.Content["answer"]; ok && v.String != nil {
				answers[ask.Prompt] = string(*v.String)
			}
		}

		return UserInputOutcome{Action: "accept", Answers: answers}
	case resp.Decline != nil:
		return UserInputOutcome{Action: "decline"}
	case resp.Cancel != nil:
		return UserInputOutcome{Action: "cancel"}
	default:
		return UserInputOutcome{Action: "decline"}
	}
}

// optionIdentity is the option's answer identity: the wire value for plan
// approval (the label when the wire omits the value — the only string the
// client could have seen), the label for ordinary asks.
func optionIdentity(interaction string, o UserInputOption) string {
	if interaction == interactionPlanApproval && o.Value != "" {
		return o.Value
	}

	return o.Label
}

// answerSelection normalizes one accepted string to the option identity:
// ordinary asks pass the label through unchanged; plan approval maps either
// string a client may return (the enum const, i.e. the wire value, or the
// display title) back to the wire value, and passes anything else through
// verbatim so the backend's allow-only-"approve" check can only grant on the
// intended literal.
func answerSelection(interaction string, options []UserInputOption, selected string) string {
	if interaction != interactionPlanApproval {
		return selected
	}

	for _, o := range options {
		identity := optionIdentity(interaction, o)
		if selected == identity || selected == o.Label {
			return identity
		}
	}

	return selected
}

// questionLine renders the question with its header as the property
// description (the form's per-field helper line).
func questionLine(q UserInputQuestion) string {
	if q.Header == "" || strings.Contains(q.Question, q.Header) {
		return q.Question
	}

	return q.Header + ": " + q.Question
}

// elicitKey is the index-keyed property name (duplicate question texts can
// never collide on the form).
func elicitKey(i int) string {
	return "q" + strconv.Itoa(i+1)
}

// strPtr is the optional-string helper (the SDK's pointer fields).
func strPtr(v string) *string {
	if v == "" {
		return nil
	}

	return &v
}
