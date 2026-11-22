package acpfront

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// The completion-side reconcile (QA-03, the second half of the field gap):
// the real desktop's turns VARY run-to-run — some stream model.streaming
// deltas (stream.go's translation covers those), some complete end_turn
// with ZERO deltas while the reply text sits in the backend's message
// store (the clean-build field turn emitted zero client frames; the
// in-repo routing tests stayed green because the deltas simply never
// arrived). On such a turn the client would see an empty reply — so when
// the machine resolves a turn COMPLETED and the turn's pump accepted zero
// message chunks, the front fetches the turn's final assistant text from
// the session/messages projection and emits it as ONE agent_message_chunk
// BEFORE the stopReason reply leaves (the flush-before-stopReason
// discipline: the chunk rides release's same flush as any streamed delta).
//
// The text source is wire truth, not guesswork (EXP-CODINGPLAN-CYCLE-01,
// poll id 24 — the completed turn's projection): result.messages[] entries
// carry info (role) + parts[]; the reply is the LAST role:"assistant"
// entry, its text in the type:"text" parts (the reasoning / step-start /
// step-finish / timeline parts are never message text). The machine's own
// idle-window polls read the same surface but their results are NOT
// reachable here: the PollSource contract wraps poll results opaquely as
// presence/time evidence — content is never interpreted, nothing is stored
// (watcher.go, RESEARCH Open Question 2 resolution) — and the watcher's
// loop is joined before Run returns. Hence ONE bounded front-side poll at
// completion, the dialect's poll shape (params {sessionId}).
//
// Honesty arms: the poll failing, the projection lacking an assistant
// entry, or the reply carrying no type:text parts means NO chunk — never a
// fabricated one — and the stopReason still leaves (the client sees the
// honest empty reply + end_turn, never a hang). Deltas DID stream (the
// pump's counter non-zero) means the reconcile never fires — no duplicate.

// reconcilePollBound bounds the reconcile's single session/messages read —
// the poll must never hold the stopReason hostage (never a hang): past the
// bound the read gives up and the honest empty reply leaves. Aligned with
// the pumpReleaseWait posture (the repo's other bounded turn-end wait);
// the capture-verbatim polls answer in milliseconds.
const reconcilePollBound = 2 * time.Second

// The messages projection's wire literals (the capture-verbatim shapes —
// EXP-CODINGPLAN-CYCLE-01 poll 24; single-word keys stay snake-safe for
// the tagliatelle contract).
const (
	assistantRole = "assistant"
	textPartType  = "text"
)

// reconcileNoDeltaReply is the Prompt turn-end hook (front.go, the
// StopCompleted case, BEFORE release's flush): a completed turn whose pump
// accepted zero message chunks gets ONE reconcile chunk with the stored
// reply text through the same turn-scoped emit the delta tap uses. The
// caller-initiated emit scopes by the slot state alone (turnActive /
// turnEmit — turnScope with an empty turnId skips the record-on-first-sight
// id check, since no envelope turnId exists to match); the slot is held
// until Prompt's deferred release, so a scoped read here is THIS turn's
// emitter by construction. Any failure arm logs structurally and returns —
// the stopReason always leaves.
func (f *Front) reconcileNoDeltaReply(ctx context.Context, sessionID string, pump *turnPump) {
	if pump.messageCount() != 0 {
		return // deltas streamed — the client has the text; reconcile would duplicate it
	}

	emit, scoped := f.turnScope("")
	if !scoped {
		return // the slot already cleared — never emit scopeless
	}

	pollCtx, cancel := context.WithTimeout(ctx, reconcilePollBound)
	defer cancel()

	res, err := f.conn.Call(pollCtx, zcode.MethodSessionMessages, map[string]any{
		wireKeySessionID: sessionID,
	})
	if err != nil {
		slog.Warn("acpfront: no-delta reconcile poll failed; replying without the stored text",
			"session", sessionID, "err", err)

		return
	}

	text := lastAssistantReplyText(res)
	if text == "" {
		slog.Warn("acpfront: no-delta reconcile found no assistant text in the messages projection; replying empty",
			"session", sessionID)

		return
	}

	// ONE chunk with the FULL stored text — it enqueues onto the same pump
	// the delta tap uses, so release's flush delivers it before the
	// stopReason (flush-before-stopReason, the QA-03 discipline).
	note := acpport.SessionNote{Kind: acpport.NoteMessage, Delta: text}
	if err := emit(note); err != nil {
		slog.Warn("acpfront: session update write failed", "err", err)
	}
}

// messagesProjection is one decoded session/messages result — the
// projection subset the extraction reads (the capture-verbatim wire
// shapes; single-word keys decode snake-safe).
type messagesProjection struct {
	Messages []projectedMessage `json:"messages"`
}

// projectedMessage is one messages[] entry: the info header (the role
// discriminator) plus the content parts.
type projectedMessage struct {
	Info struct {
		Role string `json:"role"`
	} `json:"info"`
	Parts []projectedPart `json:"parts"`
}

// projectedPart is one parts[] entry: only type:"text" members carry
// reply text (reasoning/step-*/timeline parts are never message text).
type projectedPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// lastAssistantReplyText extracts the LAST assistant message's text from a
// session/messages result: the last role:"assistant" entry's type:"text"
// parts, concatenated in part order. An undecodable result, a projection
// without an assistant entry, or a reply with no text parts answers "" —
// the unrecoverable arm, never a fabricated chunk.
func lastAssistantReplyText(res json.RawMessage) string {
	var proj messagesProjection
	if err := json.Unmarshal(res, &proj); err != nil {
		return ""
	}

	text := ""
	for i := range proj.Messages {
		if proj.Messages[i].Info.Role != assistantRole {
			continue
		}

		var reply strings.Builder

		for _, part := range proj.Messages[i].Parts {
			if part.Type == textPartType {
				reply.WriteString(part.Text)
			}
		}

		text = reply.String()
	}

	return text
}
