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
// loop is joined before Run returns. Hence bounded front-side polls at
// completion — a retry loop on a fixed interval inside a total window (the
// terminal-before-text delivery race: the first poll can read a projection
// that has not caught up yet), each read the dialect's poll shape
// (params {sessionId}).
//
// Honesty arms: the poll failing, the projection lacking an assistant
// entry, or the reply carrying no type:text parts means NO chunk — never a
// fabricated one — and the stopReason still leaves (the client sees the
// honest empty reply + end_turn, never a hang). Deltas DID stream (the
// pump's counter non-zero) means the reconcile never fires — no duplicate.

// reconcilePollBound bounds ONE session/messages read of the reconcile —
// a single read must never hold the stopReason hostage (never a hang): past
// the bound the read gives up and the retry loop decides. Aligned with the
// pumpReleaseWait posture (the repo's other bounded turn-end wait); the
// capture-verbatim polls answer in milliseconds.
const reconcilePollBound = 2 * time.Second

// The reconcile retry window (QA-03 field race, the 2026-11-23 telemetry
// run): the real desktop's terminal signal can land BEFORE the reply text
// reaches the messages projection — the first poll reads a projection that
// has not caught up yet (the field log "no-delta reconcile found no
// assistant text"). The reconcile therefore polls on a fixed interval until
// the stored reply appears or the total window closes. The window is
// generous against the observed catch-up latency (the raw captures' text
// lands within ~0.3–1s of the terminal) and bounded against a hang.
const (
	reconcileWindow       = 4 * time.Second
	reconcilePollInterval = 400 * time.Millisecond
)

// The messages projection's wire literals (the capture-verbatim shapes —
// EXP-CODINGPLAN-CYCLE-01 poll 24; single-word keys stay snake-safe for
// the tagliatelle contract).
const (
	assistantRole = "assistant"
	textPartType  = "text"
)

// reconcileNoDeltaReply is the Prompt turn-end hook (front.go, the
// StopCompleted case, BEFORE release's flush): a completed turn whose pump
// accepted zero message chunks reconciles the stored reply text through the
// same turn-scoped emit the delta tap uses — polling the session/messages
// projection on a fixed interval until the text appears or the total window
// closes (the terminal-before-text delivery race, QA-03 2026-11-23). The
// caller-initiated emit scopes by the slot state alone (turnActive /
// turnEmit — turnScope with an empty turnId skips the record-on-first-sight
// id check, since no envelope turnId exists to match); the slot is held
// until Prompt's deferred release, so a scoped read here is THIS turn's
// emitter by construction — and the held slot keeps the pump open, so a
// late text delta landing mid-window rides release's flush too (the
// message-count guard below keeps the two paths from duplicating it). Any
// failure arm logs structurally and returns — the stopReason always leaves.
func (f *Front) reconcileNoDeltaReply(ctx context.Context, sessionID string, pump *turnPump) {
	if pump.messageCount() != 0 {
		return // deltas streamed — the client has the text; reconcile would duplicate it
	}

	emit, scoped := f.turnScope("")
	if !scoped {
		return // the slot already cleared — never emit scopeless
	}

	deadline := time.Now().Add(reconcileWindow)
	for {
		text := f.reconcilePollText(ctx, sessionID)
		if text != "" {
			f.reconcileEmitStoredReply(pump, emit, text)

			return
		}

		if pump.messageCount() != 0 {
			return // late deltas are streaming — the client gets the text live
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			slog.Warn("acpfront: no-delta reconcile window closed without the stored text; replying empty",
				"session", sessionID)

			return
		}

		if !reconcileWait(ctx, remaining) {
			return // the turn's teardown owns this ctx — stop polling
		}
	}
}

// reconcilePollText performs ONE bounded session/messages read and returns
// the stored reply text — "" on a read failure or a projection without an
// assistant reply (the retry loop decides whether another attempt fits the
// window; each failure logs structurally, never aborts the loop).
func (f *Front) reconcilePollText(ctx context.Context, sessionID string) string {
	pollCtx, cancel := context.WithTimeout(ctx, reconcilePollBound)
	defer cancel()

	res, err := f.conn.Call(pollCtx, zcode.MethodSessionMessages, map[string]any{
		wireKeySessionID: sessionID,
	})
	if err != nil {
		slog.Warn("acpfront: no-delta reconcile poll failed", "session", sessionID, "err", err)

		return ""
	}

	return lastAssistantReplyText(res)
}

// reconcileEmitStoredReply emits the ONE reconcile chunk with the FULL
// stored text — unless a late delta landed mid-window (the message-count
// guard: the client already has the text, a chunk here would duplicate it).
// The chunk enqueues onto the same pump the delta tap uses, so release's
// flush delivers it before the stopReason (flush-before-stopReason, the
// QA-03 discipline).
func (f *Front) reconcileEmitStoredReply(pump *turnPump, emit func(note acpport.SessionNote) error, text string) {
	if pump.messageCount() != 0 {
		return
	}

	note := acpport.SessionNote{Kind: acpport.NoteMessage, Delta: text}
	if err := emit(note); err != nil {
		slog.Warn("acpfront: session update write failed", "err", err)
	}
}

// reconcileWait sleeps one poll interval bounded by the window remainder;
// false means the turn's ctx ended first (the teardown owns this ctx).
func reconcileWait(ctx context.Context, remaining time.Duration) bool {
	wait := reconcilePollInterval
	if wait > remaining {
		wait = remaining
	}

	select {
	case <-ctx.Done():
		return false
	case <-time.After(wait):
		return true
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
