package acpfront

import (
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// The delta tap (RESEARCH Pattern 2): the Front's face on the conn's
// ordered observer fan-out, registered BEFORE any machine (front.go New —
// registration order is delivery order, RESEARCH Pattern 3). One backend
// session/event envelope with type part.delta and field text becomes
// exactly ONE agent_message_chunk in arrival order — never synthesized,
// never an upsert (Pitfall 5). The field reasoning mapping is plan 04-04's
// RED target (agent_thought_chunk); input/output are Phase 5 tool
// surfaces. All three are silently skipped in this plan.

// observeEnvelope is the tap body; it runs on the conn's read goroutine
// (the hook contract: must not block) and routes by the front's single
// active turn.
func (f *Front) observeEnvelope(env zcode.Envelope) {
	_ = env // TODO(04-06 GREEN): part.delta field:text → sink message chunk
}
