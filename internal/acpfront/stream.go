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
// session/event envelope with type part.delta and field text becomes
// exactly ONE agent_message_chunk in arrival order — never synthesized,
// never an upsert (Pitfall 5: upserts carry cumulative part text and would
// duplicate every delta).

// part.delta kind and field literals (zcode-src schema, RESEARCH Pattern 2
// verbatim: field ∈ {"text","reasoning","input","output"}).
const (
	kindPartDelta = "part.delta"

	deltaFieldText      = "text"
	deltaFieldReasoning = "reasoning" // plan 04-04's RED target: agent_thought_chunk mapping
	deltaFieldInput     = "input"     // Phase 5 tool surface — ignored this plan
	deltaFieldOutput    = "output"    // Phase 5 tool surface — ignored this plan
)

// observeEnvelope is the tap body. It runs on the conn's read goroutine
// (the hook contract: delivery is synchronous and must not block — the tap
// parses small params and writes one chunk), AHEAD of the machine's
// ObserveEnvelope, so a terminal event can never resolve the turn ahead of
// earlier delta envelopes. Routing keys on the front's single active turn
// (the single-active-turn invariant): envelopes outside an active prompt
// turn are observation only. field reasoning/input/output are silently
// skipped this plan — the reasoning mapping is plan 04-04's RED target and
// the input/output fields are Phase 5 tool surfaces; nothing is ever
// synthesized for them (D-04 honesty).
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

	if params.Field != deltaFieldText {
		// reasoning → plan 04-04 (agent_thought_chunk, CORE-04);
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
	note := acpport.SessionNote{Kind: acpport.NoteMessage, Delta: params.Delta}
	if err := emit(note); err != nil {
		slog.Warn("acpfront: session update write failed", "session", note.SessionID, "err", err)
	}
}
