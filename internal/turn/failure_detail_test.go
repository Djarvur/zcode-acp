package turn_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// failedSessionEvent is a turn.failed carrier with an extra failure payload.
func failedSessionEvent(t *testing.T, detail map[string]any) fake.SideEffect {
	t.Helper()

	params := map[string]any{
		wireKeyType:         turn.KindTurnFailed,
		wireKeySeq:          4,
		wireKeyTurnID:       fixTurnID,
		wireKeyTraceID:      fixTraceID,
		wireKeyDeliveryKind: deliveryKind,
	}
	for key, value := range detail {
		params[key] = value
	}

	return emit(t, zcode.MethodSessionEvent, params)
}

// TestFailedTerminalDetail proves the provider's own turn.failed message rides
// Resolution.Err, so the ACP front can answer an actionable -32603 instead of
// only "turn ended: failed".
func TestFailedTerminalDetail(t *testing.T) {
	t.Parallel()

	sc := &fake.Scenario{
		Name: "turn-failed-detail",
		OnRequest: []fake.RequestRule{
			sendRule(t,
				statePatch(t, statusRunning, reasonPromptStarted),
				failedSessionEvent(t, map[string]any{
					"error": map[string]any{"message": "[1310] Weekly/Monthly Limit Exhausted"},
				}),
			),
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}

	ex := startOutcomeExchange(t, sc)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got := <-runAsync(ctx, ex.machine, "quota check")

	if got.err != nil || got.reason != turn.StopFailed {
		t.Fatalf("Run = (%q, %v), want StopFailed with no Run error", got.reason, got.err)
	}

	res := ex.machine.Resolution()
	if res.Err == nil || !strings.Contains(res.Err.Error(), "[1310] Weekly/Monthly Limit Exhausted") {
		t.Fatalf("Resolution.Err = %v, want the backend quota detail", res.Err)
	}

	ex.finish(t)
}
