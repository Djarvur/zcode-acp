package acpport

import (
	"context"

	"github.com/coder/acp-go-sdk"
)

// NewUpdateSink binds an UpdateSink to the agent-side connection: each
// note becomes exactly one session/update notification on the ACP wire —
// message notes via acp.UpdateAgentMessageText, thought notes via
// acp.UpdateAgentThoughtText (RESEARCH Pattern 2 mapping, executed-probe
// shapes). Notes are never synthesized here; the sink is a passthrough
// (D-04 honesty), and an unknown note kind fails loud instead of dropping.
func NewUpdateSink(conn *acp.AgentSideConnection) UpdateSink {
	return func(ctx context.Context, note SessionNote) error {
		notification, err := noteToNotification(note)
		if err != nil {
			return err
		}

		return conn.SessionUpdate(ctx, notification)
	}
}

// noteToNotification translates one neutral note onto the SDK's session
// update vocabulary; an unknown note kind is a loud coded error, never a
// dropped chunk.
func noteToNotification(note SessionNote) (acp.SessionNotification, error) {
	switch note.Kind {
	case NoteMessage:
		return acp.SessionNotification{
			SessionId: acp.SessionId(note.SessionID),
			Update:    acp.UpdateAgentMessageText(note.Delta),
		}, nil
	case NoteThought:
		return acp.SessionNotification{
			SessionId: acp.SessionId(note.SessionID),
			Update:    acp.UpdateAgentThoughtText(note.Delta),
		}, nil
	default:
		return acp.SessionNotification{}, NewInternal("acpport: unknown session note kind %q", note.Kind)
	}
}
