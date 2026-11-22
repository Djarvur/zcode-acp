package acpfront

// WR-02 (07-REVIEW) in-package busy-scope locks. The busy state is set
// directly on the front's unexported slot fields (turnMu/turnActive/
// turnSession) and promptModelSlash is exercised directly — the composed
// back leg (fake.Serve ⇄ zcode.Conn ⇄ Front) answers the mutating arm's
// setModel so the PROCEED arm is observable end-to-end, while the REJECT
// arm must answer before any wire frame. The external-package suite
// (slashmodel_test.go) locks the same-session busy case through the SDK
// wire surfaces with a real held turn; this file locks the scoping
// decision itself: the gate is SESSION-SCOPED (matching Cancel's WR-07
// discipline), so another session's turn on the front-wide slot never
// blocks this session's switch.
//
//	TestPromptModelSlashBusyScope — same-session busy: the CORE-05
//	  rejection rides the ONE stable turnActiveMessage surface and no
//	  setModel reaches the backend; cross-session busy: the switch
//	  proceeds (exactly one setModel, the end_turn resolution).

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	// busySID is the session the switch targets; busyOtherSID is the
	// session holding the front-wide slot in the cross-session arm.
	busySID      = "sess_busy"
	busyOtherSID = "sess_other"

	// busySnapshot is the confirmed settings snapshot the busy session's
	// switch validates against (the selector resolves against the listed
	// set — the busy rejection fires before any of that: validate-first
	// stays upstream of the slot check's subject).
	busySnapshot = `{
  "session": {"sessionId": "` + busySID + `"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash"},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "low", "defaultLevel": "low",
      "available": [{"value": "off", "label": "Off"}, {"value": "low", "label": "Low"}]
    }
  }
}`

	// busySelector is the listed providerId/modelId serialization the
	// switch applies.
	busySelector = "zai-api/GLM-5.3-Flash"
)

// busyScopeScenario scripts the mutating arm's reply; the unknown-method
// reply fails loud so a misrouted request can never pass silently (the
// corpus discipline).
func busyScopeScenario() *fake.Scenario {
	return &fake.Scenario{
		Name: "slash-busy-scope",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionSetModel, Reply: &fake.Reply{Result: json.RawMessage(
				`{"session":{"sessionId":"` + busySID + `"},"settings":{"model":` +
					`{"current":{"providerId":"zai-api","modelId":"GLM-5.3-Flash"}}}}`)},
			},
		},
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: "MethodNotFound"},
		},
		EOFDrainMs: 1,
	}
}

// holdSlot marks the front-wide slot held by holderSID (the busy state the
// gate reads; no real turn runs — promptModelSlash is exercised directly).
func holdSlot(f *Front, holderSID string) {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	f.turnActive = true
	f.turnSession = holderSID
}

// countSetModels counts the setModel frames on the recorded transcript
// (the in-package local of the external suite's transcriptIndexes).
func countSetModels(tr fake.Transcript) int {
	n := 0

	for _, line := range tr.Lines {
		if string(line.Dir) == "in" && strings.Contains(line.JSON, `"method":"session/setModel"`) {
			n++
		}
	}

	return n
}

// runBusyScopeArm drives one busy-scope arm end-to-end: it records the
// snapshot, holds the slot for holderSID, runs the /model switch for
// busySID, and returns the stop reason, the recorded transcript (the legs
// closed — the caller asserts on the wire truth), and the intercept's
// error.
func runBusyScopeArm(t *testing.T, holderSID string) (acpport.StopReason, fake.Transcript, error) {
	t.Helper()

	f, closeLegs := newInterleaveFront(t, busyScopeScenario())

	f.recordSettings(busySID, json.RawMessage(busySnapshot))
	holdSlot(f, holderSID)

	stop, handled, err := f.promptModelSlash(context.Background(), acpport.PromptParams{
		SessionID: busySID,
		Text:      "/model " + busySelector,
	}, func(context.Context, acpport.SessionNote) error { return nil })

	if !handled {
		closeLegs()
		t.Fatal("the command form must be handled by the intercept")
	}

	return stop, closeLegs(), err
}

// TestPromptModelSlashBusyScope is the WR-02 scoping lock (07-REVIEW): the
// /model intercept's busy gate is session-scoped — a switch issued while
// THIS session's turn is active takes the CORE-05 turnActiveMessage
// rejection before any backend call, while a switch on a slot-free session
// proceeds even when ANOTHER session holds the front-wide slot (the
// mutation is a conn.Call, not a turn — the Cancel WR-07 scoping parity).
func TestPromptModelSlashBusyScope(t *testing.T) {
	t.Run("same-session busy rejects on the ONE stable surface with no mutation", func(t *testing.T) {
		_, tr, err := runBusyScopeArm(t, busySID)

		if err == nil {
			t.Fatal("a same-session busy switch must be rejected (CORE-05), got success")
		}

		var seamErr *acpport.Error
		if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInvalidRequest ||
			seamErr.Message != turnActiveMessage {
			t.Fatalf("busy rejection = %v, want the coded invalid-request carrying %q", err, turnActiveMessage)
		}

		if setModels := countSetModels(tr); setModels != 0 {
			t.Fatalf("the busy rejection must precede any backend call — "+
				"%d session/setModel frame(s) recorded", setModels)
		}
	})

	t.Run("cross-session busy leaves the switch unaffected", func(t *testing.T) {
		stop, tr, err := runBusyScopeArm(t, busyOtherSID)

		if err != nil {
			t.Fatalf("a switch on a slot-free session must proceed (the gate is session-scoped), got: %v", err)
		}

		if stop != acpport.StopEndTurn {
			t.Fatalf("stop reason = %q, want end_turn", stop)
		}

		if setModels := countSetModels(tr); setModels != 1 {
			t.Fatalf("the cross-session switch must apply exactly one setModel mutation, got %d", setModels)
		}
	})
}
