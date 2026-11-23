package acpfront

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
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

// TestTurnAssistantReplyText locks the turn-scoped extraction (delta-review
// WR-01): under a start stamp only entries created no older than the stamp
// minus the skew margin qualify — a PRIOR turn's stored reply (exactly what
// a lagging projection serves) never rides as this turn's recovery; the
// stamp-less degraded arm keeps the legacy rule ONLY while unambiguous
// (at most one assistant entry); several anchor-less assistant entries are
// ambiguous by construction and answer "".
func TestTurnAssistantReplyText(t *testing.T) {
	proj := func(entry_created int64, text string) json.RawMessage {
		return json.RawMessage(`{"messages":[` +
			`{"info":{"role":"user","time":{"created":` + fmtInt(entry_created-100) + `}},"parts":[]},` +
			`{"info":{"role":"assistant","time":{"created":` + fmtInt(entry_created) + `}},` +
			`"parts":[{"type":"reasoning","text":"thinking"},{"type":"text","text":"` + text + `"}]}]}`)
	}

	const turnStamp = int64(1795338160000)
	stampMin := turnStamp - int64(reconcileTimeSkew/time.Millisecond)

	if got := turnAssistantReplyText(proj(1795338160100, "second-reply"), stampMin); got != "second-reply" {
		t.Fatalf("this turn's qualifying entry = %q, want %q", got, "second-reply")
	}

	if got := turnAssistantReplyText(proj(1795338154600, "first-reply"), stampMin); got != "" {
		t.Fatalf("a PRIOR turn's entry (created before the stamp margin) = %q, want \"\" — "+
			"the stale reply must never ride", got)
	}

	undated := json.RawMessage(`{"messages":[{"info":{"role":"assistant"},` +
		`"parts":[{"type":"text","text":"undated"}]}]}`)
	if got := turnAssistantReplyText(undated, stampMin); got != "" {
		t.Fatalf("an undated entry under a stamp = %q, want \"\" (undated entries never qualify)", got)
	}

	legacySingle := json.RawMessage(`{"messages":[{"info":{"role":"assistant"},` +
		`"parts":[{"type":"text","text":"solo"}]}]}`)
	if got := turnAssistantReplyText(legacySingle, 0); got != "solo" {
		t.Fatalf("the stamp-less single-entry arm = %q, want %q (the legacy unambiguous rule)", got, "solo")
	}

	legacyAmbiguous := json.RawMessage(`{"messages":[` +
		`{"info":{"role":"assistant"},"parts":[{"type":"text","text":"one"}]},` +
		`{"info":{"role":"assistant"},"parts":[{"type":"text","text":"two"}]}]}`)
	if got := turnAssistantReplyText(legacyAmbiguous, 0); got != "" {
		t.Fatalf("the stamp-less multi-entry arm = %q, want \"\" (ambiguous by construction)", got)
	}
}

// fmtInt is the test-local int64 formatter keeping the projection builder
// literal-safe (the repo's stdlib-only test posture).
func fmtInt(v int64) string {
	return strconv.FormatInt(v, 10)
}

// TestPumpEnqueueIfNoMessagesAtomic locks the reconcile emit's atomicity
// (delta-review WR-03): the message-count check and the enqueue share ONE
// lock acquisition — a message already accepted refuses the stored text
// with the sentinel and queues nothing; an empty pump accepts and the
// counter bumps exactly once.
func TestPumpEnqueueIfNoMessagesAtomic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sink := func(_ context.Context, _ acpport.SessionNote) error { return nil }
	p := newTurnPump(ctx, "sess_fix1", sink, cancel, 1<<20)
	defer p.releaseFlush()

	note := acpport.SessionNote{Kind: acpport.NoteMessage, Delta: "stored-reply"}

	if err := p.enqueueIfNoMessages(note); err != nil {
		t.Fatalf("enqueueIfNoMessages on an empty pump = %v, want nil (accepted)", err)
	}

	if p.messageCount() != 1 {
		t.Fatalf("message count after the accepted emit = %d, want 1", p.messageCount())
	}

	if err := p.enqueueIfNoMessages(note); !errors.Is(err, errReconcileDuplicate) {
		t.Fatalf("enqueueIfNoMessages after an accepted message = %v, want the duplicate sentinel", err)
	}

	if p.messageCount() != 1 {
		t.Fatalf("message count after the refused emit = %d, want 1 (the refusal queues nothing)", p.messageCount())
	}
}
