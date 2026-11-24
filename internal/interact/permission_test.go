package interact_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// The backend permission bridge outcome-table locks (plan 05-02 Task 2,
// T-5-04 deny-by-default): a scripted front double drives the bridge's
// Forward through every outcome arm — selected declared option (embedded
// response byte-verbatim), unknown/missing optionId, cancelled outcome,
// cancelled-classified and foreign forwarder errors, malformed params —
// with the wire vocabulary asserted per arm. Never an allow the backend's
// own options do not define.

// permParamsJSON is the canonical ask params (the corpus row's reverse
// shape): two options with embedded responses.
const permParamsJSON = `{"requestId":"perm-1","sessionId":"sess_fix1","turnId":"turn_fix1",` +
	`"toolCallId":"tc1","toolName":"Bash","reason":"run echo","riskLevel":"medium",` +
	`"input":{"command":"echo hi"},` +
	`"options":[{"optionId":"allow_once","kind":"allow_once","name":"Allow once",` +
	`"response":{"decision":"allow","reason":"Approved once"}},` +
	`{"optionId":"deny","kind":"deny","name":"Deny","response":{"decision":"deny","reason":"denied"}}]}`

// permAllowResponse is the allow_once option's embedded response bytes —
// the forward must replay them byte-verbatim.
const permAllowResponse = `{"decision":"allow","reason":"Approved once"}`

// Test-fixture literals shared across the outcome-table arms (permSelDeny
// is the deny-selection option id; responder_test's deny answers reuse it).
const (
	permFixSessionID = "sess_fix1"
	permSelDeny      = "deny"
	permSelAllowProj = "allow_project"
)

// errFakeSend is the deliberately non-cancelled foreign forwarder error.
var errFakeSend = errors.New("send pipe failed")

// permissionRequest is the normalized canonical reverse request.
func permissionRequest() interact.Request {
	return interact.Request{
		Method:    interact.MethodRequestPermission,
		ID:        &zcode.ID{Str: "p-1"},
		Params:    json.RawMessage(permParamsJSON),
		ReqID:     "perm-1",
		SessionID: permFixSessionID,
	}
}

// TestPermissionForwardArms drives the forward arms: the selected declared
// option (embedded response verbatim), the allow_always kind mapping, and
// the deny-by-default table on undefined outcomes.
func TestPermissionForwardArms(t *testing.T) {
	t.Run("selected declared option forwards embedded response verbatim", func(t *testing.T) {
		ans, err := drivePermissionArm(t, acpport.PermissionOutcome{OptionID: "allow_once"}, nil)
		assertPermissionResult(t, ans, err, permAllowResponse)
	})

	t.Run("allow_always mapping carries the backend kind", func(t *testing.T) {
		assertAllowAlwaysArm(t)
	})

	t.Run("deny-by-default on undefined outcomes", func(t *testing.T) {
		for name, outcome := range map[string]acpport.PermissionOutcome{
			"unknown optionId": {OptionID: "allow_forever"},
			"missing optionId": {},
		} {
			ans, err := drivePermissionArm(t, outcome, nil)
			assertDenyResult(t, name, ans, err)
		}
	})
}

// TestPermissionCancelledArms drives the cancelled and error arms: the ACP
// cancelled outcome, the cancelled-classified forwarder error, the foreign
// error, and malformed params (deny without forwarding).
func TestPermissionCancelledArms(t *testing.T) {
	t.Run("cancelled outcome resolves the -32021 error reply", func(t *testing.T) {
		ans, err := drivePermissionArm(t, acpport.PermissionOutcome{Cancelled: true}, nil)
		assertCancelledError(t, ans, err)
	})

	t.Run("cancelled-classified forwarder error resolves -32021", func(t *testing.T) {
		ans, err := drivePermissionArm(t, acpport.PermissionOutcome{}, context.Canceled)
		assertCancelledError(t, ans, err)
	})

	t.Run("foreign forwarder error surfaces as a bare error (-32603 at deliver)", func(t *testing.T) {
		ans, err := drivePermissionArm(t, acpport.PermissionOutcome{}, errFakeSend)
		if err == nil {
			t.Fatalf("foreign forwarder error must surface, got answer %+v", ans)
		}

		if ans.Err != nil || ans.Result != nil {
			t.Fatalf("foreign error must be a bare error (deliver wraps -32603), got %+v", ans)
		}
	})

	t.Run("malformed params with no usable options deny without forwarding", func(t *testing.T) {
		assertMalformedParamsArm(t)
	})
}

// TestPermissionAskNormalization digs the canonical correlation keys and
// the mapped option kinds the bridge hands the front.
func TestPermissionAskNormalization(t *testing.T) {
	var captured acpport.PermissionAsk

	_, _ = drivePermissionArmCapture(t, &captured, acpport.PermissionOutcome{OptionID: permSelDeny}, nil)

	if captured.SessionID != permFixSessionID || captured.RequestID != "perm-1" ||
		captured.ToolCallID != "tc1" || captured.ToolName != "Bash" {
		t.Fatalf("captured ask = %+v, want the canonical correlation keys", captured)
	}

	if len(captured.Options) != 2 {
		t.Fatalf("captured ask options = %d entries, want 2", len(captured.Options))
	}

	if captured.Options[0].Kind != acpport.PermKindAllowOnce || captured.Options[1].Kind != acpport.PermKindRejectOnce {
		t.Fatalf("mapped kinds = %q/%q, want allow_once/reject_once (the locked table)",
			captured.Options[0].Kind, captured.Options[1].Kind)
	}
}

// assertAllowAlwaysArm checks the allow_project option maps onto the ACP
// allow_always kind with optionId/name verbatim.
func assertAllowAlwaysArm(t *testing.T) {
	t.Helper()

	params := json.RawMessage(`{"requestId":"perm-2","sessionId":"sess_fix1","toolCallId":"tc2",` +
		`"options":[{"optionId":"allow_project","kind":"allow_project","name":"Allow for project",` +
		`"response":{"decision":"allow","reason":"project"}}]}`)

	var captured acpport.PermissionAsk

	bridge := interact.NewPermissionBridge(recordingFront(
		t, &captured, acpport.PermissionOutcome{OptionID: permSelAllowProj}, nil))

	ans, err := bridge.Forward(armCtx(), interact.Request{
		Method:    interact.MethodRequestPermission,
		ID:        &zcode.ID{Str: "p-2"},
		Params:    params,
		ReqID:     "perm-2",
		SessionID: permFixSessionID,
	})

	assertPermissionResult(t, ans, err, `{"decision":"allow","reason":"project"}`)

	if len(captured.Options) != 1 || captured.Options[0].Kind != acpport.PermKindAllowAlways ||
		captured.Options[0].OptionID != permSelAllowProj || captured.Options[0].Name != "Allow for project" {
		t.Fatalf("mapped ask options = %+v, want kind allow_always with optionId/name verbatim", captured.Options)
	}
}

// assertMalformedParamsArm checks an options-less ask denies WITHOUT
// forwarding (the front double must never run).
func assertMalformedParamsArm(t *testing.T) {
	t.Helper()

	calls := 0

	bridge := interact.NewPermissionBridge(
		func(_ context.Context, _ acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
			calls++

			return acpport.PermissionOutcome{}, nil
		})

	ans, err := bridge.Forward(armCtx(), interact.Request{
		Method: interact.MethodRequestPermission,
		ID:     &zcode.ID{Str: "p-3"},
		Params: json.RawMessage(`{"requestId":"perm-3","sessionId":"sess_fix1"}`), // no options array
		ReqID:  "perm-3",
	})

	assertDenyResult(t, "no options array", ans, err)

	if calls != 0 {
		t.Fatalf("front bridge ran %d times for an options-less ask, want 0 (deny without forwarding)", calls)
	}
}

// armCtx is the context one outcome-arm drive runs on — the arms complete
// synchronously, so the background context suffices.
func armCtx() context.Context {
	return context.Background()
}

// drivePermissionArm runs one Forward over the canonical ask with the
// scripted outcome.
func drivePermissionArm(t *testing.T, outcome acpport.PermissionOutcome, err error) (interact.Answer, error) {
	t.Helper()

	return drivePermissionArmCapture(t, nil, outcome, err)
}

// drivePermissionArmCapture additionally captures the ask the bridge
// hands the front double.
func drivePermissionArmCapture(
	t *testing.T, captured *acpport.PermissionAsk, outcome acpport.PermissionOutcome, frontErr error,
) (interact.Answer, error) {
	t.Helper()

	bridge := interact.NewPermissionBridge(
		func(_ context.Context, ask acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
			if captured != nil {
				*captured = ask
			}

			return outcome, frontErr
		})

	return bridge.Forward(armCtx(), permissionRequest())
}

// recordingFront builds a front double that records the ask.
func recordingFront(
	t *testing.T, captured *acpport.PermissionAsk, outcome acpport.PermissionOutcome, frontErr error,
) interact.FrontBridge {
	t.Helper()

	return func(_ context.Context, ask acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
		*captured = ask

		return outcome, frontErr
	}
}

// assertPermissionResult asserts a success answer carrying the given
// result bytes verbatim.
func assertPermissionResult(t *testing.T, ans interact.Answer, err error, wantRaw string) {
	t.Helper()

	if err != nil {
		t.Fatalf("forward must succeed, got error: %v", err)
	}

	if ans.Err != nil {
		t.Fatalf("forward must answer a result, got wire error %+v", ans.Err)
	}

	raw, ok := ans.Result.(json.RawMessage)
	if !ok {
		t.Fatalf("result = %T (%v), want json.RawMessage passthrough", ans.Result, ans.Result)
	}

	if compactJSON(t, raw) != compactJSON(t, json.RawMessage(wantRaw)) {
		t.Fatalf("forwarded response = %s, want the option's embedded response %s (verbatim)", raw, wantRaw)
	}
}

// assertDenyResult asserts the deny-by-default answer: decision deny with
// the denied-by-user reason.
func assertDenyResult(t *testing.T, name string, ans interact.Answer, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("deny arm must answer a result, got error: %v", err)
	}

	if ans.Err != nil {
		t.Fatalf("deny arm must answer a RESULT (never an error reply), got %+v", ans.Err)
	}

	decision, ok := ans.Result.(map[string]any)
	if !ok || decision["decision"] != permSelDeny || decision["reason"] != "Permission denied by user" {
		t.Fatalf("%s: deny arm answer = %v, want {decision:deny, reason:denied-by-user}", name, ans.Result)
	}
}

// assertCancelledError asserts the -32021 error reply with the stable
// cancelled message.
func assertCancelledError(t *testing.T, ans interact.Answer, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("cancelled arm must answer a coded error reply, got error: %v", err)
	}

	if ans.Result != nil {
		t.Fatalf("cancelled arm must not carry a result (never a fabricated decision), got %v", ans.Result)
	}

	if ans.Err == nil || ans.Err.Code != zcode.CodeHostRequestCancelled {
		t.Fatalf("cancelled arm error = %+v, want code %d (-32021)", ans.Err, zcode.CodeHostRequestCancelled)
	}

	if ans.Err.Message != "Client request cancelled: interaction/requestPermission" {
		t.Fatalf("cancelled arm message = %q, want the stable constant", ans.Err.Message)
	}
}

// TestDenyPermissionForwarder locks the text-mode permission posture: a
// stray ask answers exactly the wrapper-synthesized deny — no option id, no
// embedded response, no params content is echoed (the answer map IS the
// whole reply payload).
func TestDenyPermissionForwarder(t *testing.T) {
	ans, err := interact.DenyPermissionForwarder(armCtx(), permissionRequest())
	if err != nil {
		t.Fatalf("DenyPermissionForwarder must answer a result, got error: %v", err)
	}

	if ans.Err != nil {
		t.Fatalf("DenyPermissionForwarder must never answer a wire error, got %+v", ans.Err)
	}

	want := map[string]any{"decision": permSelDeny, "reason": "Permission denied by user"}

	decision, ok := ans.Result.(map[string]any)
	if !ok {
		t.Fatalf("result = %T (%v), want the deny decision map", ans.Result, ans.Result)
	}

	if !reflect.DeepEqual(decision, want) {
		t.Fatalf("deny result = %v, want exactly %v (no option id or params echo)", decision, want)
	}
}

// compactJSON normalizes two raw payloads for a byte-order-insensitive
// compare (the reply envelope marshals through the codec, which compacts).
func compactJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode raw payload %s: %v", raw, err)
	}

	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("re-marshal payload: %v", err)
	}

	return string(out)
}
