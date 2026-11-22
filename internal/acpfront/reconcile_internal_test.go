package acpfront

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestLastAssistantReplyText locks the reconcile's extraction on the
// capture-verbatim projection shapes (EXP-CODINGPLAN-CYCLE-01 poll 24 —
// the completed turn's reply): the LAST role:"assistant" entry's
// type:"text" parts concatenated in part order; reasoning/step-*/timeline
// parts and user entries never ride; undecodable results and
// assistant-less projections answer "" — the unrecoverable arm.
func TestLastAssistantReplyText(t *testing.T) {
	for _, tt := range lastReplyTextCases() {
		t.Run(tt.name, func(t *testing.T) {
			if got := lastAssistantReplyText(tt.res); got != tt.want {
				t.Fatalf("lastAssistantReplyText = %q, want %q", got, tt.want)
			}
		})
	}
}

// lastReplyTextCase is one extraction-table row.
type lastReplyTextCase struct {
	name string
	res  json.RawMessage
	want string
}

// lastReplyTextCases builds the full extraction table: the extraction-shape
// rows plus the empty-arm rows (the split keeps both builders readable).
func lastReplyTextCases() []lastReplyTextCase {
	return append(lastReplyTextShapeCases(), lastReplyTextEmptyCases()...)
}

// lastReplyTextShapeCases assembles capture-shape projection fixtures —
// raw JSON with the wire keys verbatim — for the extraction rows.
func lastReplyTextShapeCases() []lastReplyTextCase {
	assistant := func(parts string) string {
		return `{"info":{"role":"assistant"},"parts":` + parts + `}`
	}

	user := func(text string) string {
		return `{"info":{"role":"user"},"parts":[{"type":"text","text":"` + text + `"}]}`
	}

	projection := func(messages ...string) json.RawMessage {
		return json.RawMessage(`{"messages":[` + strings.Join(messages, ",") + `]}`)
	}

	return []lastReplyTextCase{
		{
			name: "capture shape: timeline assistant, user, reply assistant (reasoning+text+step parts)",
			res: projection(
				`{"info":{"role":"assistant"},"parts":[{"type":"timeline","display":"separator"}]}`,
				user("Reply with the single word: ok"),
				assistant(`[`+
					`{"type":"step-start"},`+
					`{"type":"reasoning","text":"thinking"},`+
					`{"type":"text","text":"ok"},`+
					`{"type":"step-finish","reason":"stop"}]`),
			),
			want: "ok",
		},
		{
			name: "last assistant wins over an earlier text-bearing assistant",
			res: projection(
				assistant(`[{"type":"text","text":"earlier"}]`),
				user("next"),
				assistant(`[{"type":"text","text":"reply"}]`),
			),
			want: "reply",
		},
		{
			name: "multi text parts concatenate in part order",
			res: projection(
				assistant(`[{"type":"text","text":"Answer:"},{"type":"text","text":" done"}]`),
			),
			want: "Answer: done",
		},
	}
}

// lastReplyTextEmptyCases pins the unrecoverable arms: every shape that
// must answer "" — never a fabricated chunk.
func lastReplyTextEmptyCases() []lastReplyTextCase {
	return []lastReplyTextCase{
		{
			name: "assistant with only reasoning parts answers empty",
			res: json.RawMessage(`{"messages":[{"info":{"role":"assistant"},` +
				`"parts":[{"type":"reasoning","text":"thinking only"}]}]}`),
			want: "",
		},
		{
			name: "no assistant entries answers empty",
			res: json.RawMessage(`{"messages":[{"info":{"role":"user"},` +
				`"parts":[{"type":"text","text":"prompt only"}]}]}`),
			want: "",
		},
		{
			name: "empty messages answers empty",
			res:  json.RawMessage(`{"messages":[],"sessionId":"sess_fix1"}`),
			want: "",
		},
		{
			name: "missing messages key answers empty",
			res:  json.RawMessage(`{}`),
			want: "",
		},
		{
			name: "undecodable result answers empty",
			res:  json.RawMessage(`{"messages":not-json`),
			want: "",
		},
		{
			name: "empty result answers empty",
			res:  json.RawMessage(``),
			want: "",
		},
	}
}
