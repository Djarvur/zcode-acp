package acpfront

import (
	"encoding/json"
	"log/slog"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// The delta tap (RESEARCH Pattern 2): the Front's face on the conn's
// ordered observer fan-out, registered BEFORE any machine (front.go New —
// registration order is delivery order, RESEARCH Pattern 3). One backend
// session/event envelope with type part.delta becomes exactly ONE chunk in
// arrival order — field text maps to agent_message_chunk, field reasoning
// to agent_thought_chunk (CORE-03/CORE-04) — never synthesized, never an
// upsert (Pitfall 5: upserts carry cumulative part text and would
// duplicate every delta).

// part.delta kind and field literals (zcode-src schema, RESEARCH Pattern 2
// verbatim: field ∈ {"text","reasoning","input","output"}).
const (
	kindPartDelta = "part.delta"

	deltaFieldText      = "text"      // → agent_message_chunk (acpport.NoteMessage)
	deltaFieldReasoning = "reasoning" // → agent_thought_chunk (acpport.NoteThought, CORE-04)
	deltaFieldInput     = "input"     // Phase 5 tool surface — ignored, never synthesized
	deltaFieldOutput    = "output"    // Phase 5 tool surface — ignored, never synthesized
)

// observeEnvelope is the tap body. It runs on the conn's read goroutine
// (the hook contract: delivery is synchronous and must not block — the tap
// parses small params and writes one chunk), AHEAD of the machine's
// ObserveEnvelope, so a terminal event can never resolve the turn ahead of
// earlier delta envelopes. Routing keys on the front's single active turn
// (the single-active-turn invariant): envelopes outside an active prompt
// turn are observation only. field input/output are Phase 5 tool surfaces
// and are silently skipped — nothing is ever synthesized for them (D-04
// honesty).
func (f *Front) observeEnvelope(env zcode.Envelope) {
	if env.Method != zcode.MethodSessionEvent {
		return
	}

	var params struct {
		Type  string `json:"type"`
		Field string `json:"field"`
		Delta string `json:"delta"`
	}

	if err := json.Unmarshal(env.Params, &params); err != nil || params.Type != kindPartDelta {
		return
	}

	var kind acpport.SessionNoteKind

	switch params.Field {
	case deltaFieldText:
		kind = acpport.NoteMessage
	case deltaFieldReasoning:
		// CORE-04: backend reasoning deltas are the ONLY source of thought
		// chunks — the acpport sink maps the note onto
		// acp.UpdateAgentThoughtText (the executed-probe wire shape).
		kind = acpport.NoteThought
	default:
		// input/output → Phase 5 tool surfaces. Skipped, never synthesized.
		return
	}

	f.turnMu.Lock()
	emit := f.turnEmit
	f.turnMu.Unlock()

	if emit == nil {
		return // no active prompt turn — pre-turn notifications are observation only
	}

	// Exactly ONE chunk per part.delta envelope, in arrival order, on the
	// turn context (alive until the prompt returns — post-cancel drain
	// still reaches the client, RESEARCH Pattern 5). The emitter binds the
	// active turn's session id and context; a failed update write (a
	// stalled or closing ACP wire) is recorded structurally — never with
	// params/content — and the turn continues; the prompt answer carries
	// the honest outcome.
	note := acpport.SessionNote{Kind: kind, Delta: params.Delta}
	if err := emit(note); err != nil {
		slog.Warn("acpfront: session update write failed", "session", note.SessionID, "err", err)
	}
}
