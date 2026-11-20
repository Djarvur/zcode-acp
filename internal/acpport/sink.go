package acpport

import (
	"context"

	"github.com/coder/acp-go-sdk"
)

// NewUpdateSink binds an UpdateSink to the agent-side connection: each
// note becomes exactly one session/update notification on the ACP wire —
// message notes via acp.UpdateAgentMessageText, thought notes via
// acp.UpdateAgentThoughtText (RESEARCH Pattern 2 mapping). Notes are
// never synthesized here; the sink is a passthrough (D-04 honesty).
func NewUpdateSink(conn *acp.AgentSideConnection) UpdateSink {
	return func(ctx context.Context, note SessionNote) error {
		return nil // RED stub (04-02 TDD): chunk translation lands in GREEN.
	}
}
