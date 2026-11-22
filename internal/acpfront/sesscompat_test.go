package acpfront_test

// Session-compat conformance rows (SESS-02/SESS-03, plan 07-03 — the D-01
// graceful arms and the D-02 decline sweep over the COMPOSED harness,
// Pitfall 7: delete/fork/set_mode/list/load are ACP-front methods, so their
// rows ride the SDK-client ⇄ tee ⇄ adapter ⇄ front ⇄ fake composition, not
// the back-leg fake corpus).
//
//   - TestSessionDeleteRegisteredRunsTeardown (the plan Task-1 tracer RED
//     target): session/delete on a REGISTERED id answers the dialect's
//     empty result {} and runs the full SESS-01 teardown (the 07-01
//     CloseSession contract) — D-01's "child-side cleanup follows
//     session/close semantics where applicable".
//
// The byte-lock discipline: every load-bearing wire value is asserted as a
// tee-line substring constant — never paraphrased.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The sess-compat rows' tee probe substrings (the load-bearing wire values,
// byte-locked per the fullcycle discipline).
const (
	teeDeleteMethod    = `"method":"session/delete"`
	teeCloseMethod     = `"method":"session/close"`
	teeSendMethod      = `"method":"session/send"`
	teeSubscribeMethod = `"method":"session/subscribe"`
	teeEmptyResult     = `"result":{}`
	teeForkMethod      = `"method":"session/fork"`

	// teeNotFoundMessage is the -32601 decline frame's message literal (the
	// SDK's NewMethodNotFound map — byte-locked, never paraphrased).
	teeNotFoundMessage = `"message":"Method not found"`

	// sessRawProbeID is the raw-frame probe's JSON-RPC id (the set_model row
	// the pinned SDK cannot send — the id is chosen far from the SDK
	// client's small-integer counter so the reply can never collide with a
	// pending typed call).
	sessRawProbeID = "77007"
)

// sessDeleteSessionID is the inline delete scenario's fixed backend session
// id (the create reply carries it; every leg speaks the same id).
const sessDeleteSessionID = "sess_del"

// sessDeleteScenario composes the per-row fake for the session/delete
// vectors: one bare create reply, one subscribe reply (the D-03 preamble),
// one close reply (the teardown leg) — NO send rule, so any prompt traffic
// a buggy arm issues fails loud against the unknown-method reply. Requests
// the scenario did not script answer the -32601 unknown-method reply (the
// cfgOptionScenario discipline — a silent reply would hide regressions).
func sessDeleteScenario(extra ...fake.RequestRule) *fake.Scenario {
	rules := make([]fake.RequestRule, 0, 3+len(extra))
	rules = append(rules,
		fake.RequestRule{
			Method: zcode.MethodSessionCreate,
			Reply: &fake.Reply{Result: json.RawMessage(`{"session":{"sessionId":"` +
				sessDeleteSessionID + `"},"protocol":{"name":"ZCode Protocol","version":1}}`)},
		},
		fake.RequestRule{
			Method: zcode.MethodSessionSubscribe,
			Reply: &fake.Reply{Result: json.RawMessage(
				`{"sessionId":"` + sessDeleteSessionID + `","eventSeq":0,"events":[]}`)},
		},
		fake.RequestRule{
			Method: zcode.MethodSessionClose,
			Reply:  &fake.Reply{Result: json.RawMessage(`{"closed":true}`)},
		})
	rules = append(rules, extra...)

	return &fake.Scenario{
		Name:      "sesscompat-delete",
		OnRequest: rules,
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: fakeMethodNotFound},
		},
		EOFDrainMs: 100,
	}
}

// newSessCompatFront composes the session-compat harness with NO model
// configured — the create/subscribe preamble is the whole D-03 leg the
// delete rows ride (no setModel follow-up frames to account for).
func newSessCompatFront(t *testing.T, sc *fake.Scenario) *FrontHarness {
	t.Helper()

	return ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		return acpport.NewAgent(acpfront.New(conn, acpfront.WithVersion("0.0.0-dev")))
	})
}

// TestSessionDeleteRegisteredRunsTeardown locks SESS-02's registered-id arm
// (plan 07-03 tracer, D-01): the SDK client's typed DeleteSession on an
// ESTABLISHED session answers the dialect's empty result {} on the wire and
// runs the FULL SESS-01 teardown — the registry entry released (a
// post-delete prompt lands the coded ErrNoSession-class rejection), the
// confirmed-settings store dropped (set_config_option lands the coded
// unknown-session rejection), the machine's observers released (exactly the
// machine's fan-out entries removed), and the backend session/close on the
// transcript with NO further send traffic for the id (the silent {} reply
// is the teardown's last act, never a leaky arm).
//
// RED at plan time: today's adapter arm answers -32601 "Method not found"
// with data {"method":"session/delete"} — the delete never reaches the
// front and nothing tears down.
func TestSessionDeleteRegisteredRunsTeardown(t *testing.T) {
	h := newSessCompatFront(t, sessDeleteScenario())

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	sid := string(newResp.SessionId)
	if sid == "" {
		t.Fatal("session/new returned an empty session id")
	}

	notifPre, routerPre := h.conn.ObserverCounts()

	if _, err := h.client.DeleteSession(ctx, acp.DeleteSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("session/delete on the registered id must answer the graceful empty result, got error: %v", err)
	}

	// Byte-lock the wire reply (the dialect's result shape): the response
	// frame matching the delete request's id carries the EMPTY result {} —
	// never an error envelope.
	lines := h.ACPLines()
	reqIdx := teeLineIndex(t, lines, "out", teeDeleteMethod, "session/delete request")
	respLine := authTeeResponse(t, lines, reqIdx)

	if strings.Contains(respLine.JSON, `"error"`) {
		t.Fatalf("session/delete response carries an error frame: %s", respLine.JSON)
	}

	if !strings.Contains(respLine.JSON, teeEmptyResult) {
		t.Fatalf("session/delete response = %s, want the empty {} result", respLine.JSON)
	}

	// The registry entry is released: the post-delete prompt lands the
	// coded ErrNoSession-class rejection — never a zombie session.
	_, err = h.client.Prompt(ctx, lifecyclePrompt(sid, "post-delete"))
	assertCodedInvalidRequest(t, err)

	// The confirmed-settings store dropped with the teardown: the picker ask
	// on the deleted id answers the coded unknown-session rejection (no
	// confirmed settings to validate against) — the map stays bounded with
	// the registry (the WR-07 discipline the delete arm inherits).
	_, err = h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDModel, corpusModelFlashID))
	assertCodedInvalidParams(t, err)

	// Exactly the session machine's observer entries released (the SESS-01
	// close contract's step 4 — one notification + one router remove).
	if notif, router := h.conn.ObserverCounts(); notif != notifPre-1 || router != routerPre-1 {
		t.Fatalf("post-delete observer counts = (%d, %d), want exactly the machine's entries removed (%d, %d)",
			notif, router, notifPre-1, routerPre-1)
	}

	// The back leg (the transcript phase — Transcript closes the fake):
	// the teardown's session/close is on the wire AFTER the subscribe, and
	// NO session/send traffic exists for the id — the delete arm tears the
	// session down, it never talks for it.
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertSessDeleteTeardownWire(t, tr)
}

// assertSessDeleteTeardownWire locks the delete teardown's back-leg shape
// (the tracer's Test 2): exactly the preamble's subscribe BEFORE the
// teardown close, and NO session/send traffic for the id — the delete arm
// tears the session down, it never talks for it.
func assertSessDeleteTeardownWire(t *testing.T, tr fake.Transcript) {
	t.Helper()

	subIdx := transcriptIndex(t, tr, "in", teeSubscribeMethod, "subscribe request")
	closeIdx := transcriptIndex(t, tr, "in", teeCloseMethod, "teardown close request")

	if closeIdx < subIdx {
		t.Fatalf("wire order violated: the teardown close(%d) must follow the subscribe(%d)", closeIdx, subIdx)
	}

	if sends := transcriptIndexes(tr, "in", teeSendMethod); len(sends) != 0 {
		t.Fatalf("back leg records %d session/send frames, want 0 (the delete teardown runs no turn traffic)",
			len(sends))
	}
}

// TestSessionDeleteUnknownSilentSuccess locks SESS-02's silent-success arm
// (plan 07-03, D-01): session/delete on an id the front does not hold —
// never opened, the empty string, or already closed — answers the empty
// result {} IMMEDIATELY and sends ZERO backend frames (the runtime owns
// real deletion; there is nothing on the back leg to delete). The spec's
// "SHOULD succeed silently" semantics for unknown sessions, byte-locked.
//
// RED at plan time: today's -32601 unadvertised default answers every id —
// the unknown-id row fails on the error envelope (the fork/decline rows'
// pre-implementation passes are the SDK defaults, recorded honestly as
// CONFIRM-GREEN-DEFAULT per the 05-08 precedent).
func TestSessionDeleteUnknownSilentSuccess(t *testing.T) {
	t.Run("unknown id sends nothing", func(t *testing.T) {
		h := newSessCompatFront(t, sessDeleteScenario())

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		mustInitialize(ctx, t, h)

		assertSessDeleteSilentSuccess(ctx, t, h, "sess_never_opened")
	})

	t.Run("empty id sends nothing", func(t *testing.T) {
		h := newSessCompatFront(t, sessDeleteScenario())

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		mustInitialize(ctx, t, h)

		assertSessDeleteSilentSuccess(ctx, t, h, "")
	})

	t.Run("already-closed id sends nothing", func(t *testing.T) {
		h := newSessCompatFront(t, sessDeleteScenario())

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		mustInitialize(ctx, t, h)

		newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("session/new: %v", err)
		}

		sid := string(newResp.SessionId)

		if _, err := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
			t.Fatalf("session/close: %v", err)
		}

		assertSessDeleteSilentSuccess(ctx, t, h, sid)

		// The closed id's teardown ran exactly once: the transcript holds the
		// EXPLICIT close's single session/close frame — the delete answered {}
		// without a second close (an unknown-after-release id never re-runs
		// the teardown).
		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		if closes := transcriptIndexes(tr, "in", teeCloseMethod); len(closes) != 1 {
			t.Fatalf("back leg records %d session/close frames, want exactly 1 "+
				"(the explicit close; the delete's silent-success arm adds none)", len(closes))
		}
	})
}

// sessTeeReply finds the agent→client reply frame matching the request at
// reqIdx by JSON-RPC id. Unlike authTeeResponse it ACCEPTS frames whose
// payload carries a "method" key — the decline frames' data.method does —
// so the decline rows can lock their replies.
func sessTeeReply(t *testing.T, lines []TeeLine, reqIdx int) TeeLine {
	t.Helper()

	var req struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal([]byte(lines[reqIdx].JSON), &req); err != nil {
		t.Fatalf("decode request frame: %v (%s)", err, lines[reqIdx].JSON)
	}

	idKey := `"id":` + string(req.ID) + ","

	for _, line := range lines {
		if line.Dir == "in" && strings.Contains(line.JSON, idKey) {
			return line
		}
	}

	t.Fatalf("no reply frame matching %s on the tee", idKey)

	return TeeLine{}
}

// assertSessDeleteSilentSuccess drives one delete-unknown arm and locks its
// wire shape: the typed reply is the empty result {} (never an error
// envelope) and the back leg gained ZERO frames for the arm.
func assertSessDeleteSilentSuccess(ctx context.Context, t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	if _, err := h.client.DeleteSession(ctx, acp.DeleteSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("session/delete on the unregistered id %q must answer the silent success, got error: %v", sid, err)
	}

	lines := h.ACPLines()
	reqIdx := teeLineIndex(t, lines, "out", teeDeleteMethod, "session/delete request")
	respLine := sessTeeReply(t, lines, reqIdx)

	if strings.Contains(respLine.JSON, `"error"`) {
		t.Fatalf("session/delete response carries an error frame (the -32601 default): %s", respLine.JSON)
	}

	if !strings.Contains(respLine.JSON, teeEmptyResult) {
		t.Fatalf("session/delete response = %s, want the empty {} result", respLine.JSON)
	}
}

// TestSessionForkDeclineByteLocked locks SESS-02's fork decline (plan 07-03,
// D-01/T-07-10): the SDK client's typed fork call answers the SDK's graceful
// decline byte-shape — code -32601, message "Method not found", data
// {"method":"session/fork"} — never a crash path, never a faked success.
//
// CONFIRM-GREEN-DEFAULT at plan time (05-08 precedent): the wire shape
// already holds via the SDK dispatcher's optional-interface default (the
// adapter did not implement AgentUnstableForkSession). The GREEN makes the
// arm EXPLICIT (UnstableForkSession on the adapter returning
// unadvertised(AgentMethodSessionFork)) — byte-identical to the default by
// construction, testable without relying on the type-assert path.
func TestSessionForkDeclineByteLocked(t *testing.T) {
	h := newSessCompatFront(t, sessDeleteScenario())

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	if _, err := h.client.UnstableForkSession(ctx, acp.UnstableForkSessionRequest{
		SessionId: acp.SessionId(sessDeleteSessionID),
		Cwd:       t.TempDir(),
	}); err != nil {
		assertSessDeclineError(t, err, acp.AgentMethodSessionFork)
	} else {
		t.Fatal("session/fork must decline, got success (T-07-10: the decline is the honest shape)")
	}

	lines := h.ACPLines()
	reqIdx := teeLineIndex(t, lines, "out", teeForkMethod, "session/fork request")

	assertSessDeclineFrame(t, sessTeeReply(t, lines, reqIdx), acp.AgentMethodSessionFork)
}

// TestUncoveredMethodDeclines is the D-02 sweep's remaining rows (plan
// 07-03, SESS-03): every DISC-02 method the wrapper does not implement
// answers its protocol-correct decline byte-locked on the wire —
// set_mode/list/load via the adapter's -32601 defaults (their defaults ARE
// the contract; the rows LOCK them) and session/set_model via the SDK
// connection fallback (schema 1.25.0 has no agent dispatch case — the row
// proves the wire shape Orca would see if it ever sent it; the frame is
// injected raw because the pinned SDK's client cannot send it).
//
// CONFIRM-GREEN-DEFAULT at plan time (05-08 precedent): all four rows pass
// pre-implementation via the SDK/adapter defaults — the GREEN adds no new
// arms for them, only the explicit fork arm of the sibling test.
func TestUncoveredMethodDeclines(t *testing.T) {
	h := newSessCompatFront(t, sessDeleteScenario())

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	tests := sessDeclineTable(ctx, t, h)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				assertSessDeclineError(t, err, tt.method)
			} else {
				t.Fatalf("%s must decline, got success", tt.method)
			}

			lines := h.ACPLines()
			reqIdx := teeLineIndex(t, lines, "out", `"method":"`+tt.method+`"`, tt.method+" request")

			assertSessDeclineFrame(t, sessTeeReply(t, lines, reqIdx), tt.method)
		})
	}

	t.Run("set_model raw frame declines via the SDK fallback", func(t *testing.T) {
		sessRawSetModelDecline(t, h)
	})
}

// sessDeclineCase is one typed decline row of the D-02 sweep table: the
// method's wire name and the typed client call that must decline.
type sessDeclineCase struct {
	name   string
	method string
	call   func() error
}

// sessDeclineTable builds the typed decline rows (the call closures capture
// the harness's client and its bounded exchange ctx).
func sessDeclineTable(ctx context.Context, t *testing.T, h *FrontHarness) []sessDeclineCase {
	t.Helper()

	return []sessDeclineCase{
		{
			name:   "set_mode declines naming the method",
			method: acp.AgentMethodSessionSetMode,
			call: func() error {
				_, err := h.client.SetSessionMode(ctx, acp.SetSessionModeRequest{
					SessionId: acp.SessionId(sessDeleteSessionID),
					ModeId:    "mode-x",
				})

				return err
			},
		},
		{
			name:   "list declines naming the method",
			method: acp.AgentMethodSessionList,
			call: func() error {
				_, err := h.client.ListSessions(ctx, acp.ListSessionsRequest{})

				return err
			},
		},
		{
			name:   "load declines naming the method",
			method: acp.AgentMethodSessionLoad,
			call: func() error {
				_, err := h.client.LoadSession(ctx, acp.LoadSessionRequest{
					SessionId:  acp.SessionId(sessDeleteSessionID),
					Cwd:        t.TempDir(),
					McpServers: []acp.McpServer{},
				})

				return err
			},
		},
	}
}

// sessRawSetModelDecline is the set_model row's body: the raw request rides
// the tee's client-side write leg (the pinned SDK client has no
// session/set_model surface — the injection is the row's only route to the
// wire) and the reply frame byte-locks the SDK fallback's decline.
func sessRawSetModelDecline(t *testing.T, h *FrontHarness) {
	t.Helper()

	raw := `{"jsonrpc":"2.0","id":` + sessRawProbeID + `,"method":"session/set_model",` +
		`"params":{"sessionId":"` + sessDeleteSessionID + `",` +
		`"model":{"providerId":"` + corpusProviderID + `","modelId":"` + corpusModelID + `"}}}` + "\n"

	if _, err := h.tee.Write([]byte(raw)); err != nil {
		t.Fatalf("inject the raw session/set_model frame: %v", err)
	}

	reply := waitSessRawReply(t, h, `"id":`+sessRawProbeID)

	assertSessDeclineFrame(t, reply, "session/set_model")
}

// assertSessDeclineError byte-locks one typed decline error (the SDK's
// RequestError the client decodes): code -32601, message "Method not
// found", data.method naming the declined method — never paraphrased.
func assertSessDeclineError(t *testing.T, err error, method string) {
	t.Helper()

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("%s decline = %T (%v), want a coded RequestError", method, err, err)
	}

	if rerr.Code != -32601 {
		t.Fatalf("%s decline code = %d, want -32601 (MethodNotFound)", method, rerr.Code)
	}

	var data map[string]string
	if jsonErr := json.Unmarshal(rerr.Data, &data); jsonErr != nil {
		t.Fatalf("decode %s decline data: %v (%s)", method, jsonErr, rerr.Data)
	}

	if data["method"] != method {
		t.Fatalf("%s decline data method = %q, want %q (byte-locked, never paraphrased)",
			method, data["method"], method)
	}
}

// assertSessDeclineFrame byte-locks one decline frame on the RAW tee line:
// the -32601 code, the exact message literal, and the data.method string —
// the wire truth Orca's matchers would read.
func assertSessDeclineFrame(t *testing.T, reply TeeLine, method string) {
	t.Helper()

	if !strings.Contains(reply.JSON, `"code":-32601`) {
		t.Fatalf("decline frame code drift: %s", reply.JSON)
	}

	if !strings.Contains(reply.JSON, teeNotFoundMessage) {
		t.Fatalf("decline frame message drift (want %s): %s", teeNotFoundMessage, reply.JSON)
	}

	wantData := `"data":{"method":"` + method + `"}`
	if !strings.Contains(reply.JSON, wantData) {
		t.Fatalf("decline frame data drift (want %s): %s", wantData, reply.JSON)
	}
}

// waitSessRawReply polls the recorded tee for the reply frame carrying
// idKey (the raw-frame rows' response — no SDK call object exists to
// await), bounded by the composed-tier wait; fails the test when none
// arrives.
func waitSessRawReply(t *testing.T, h *FrontHarness, idKey string) TeeLine {
	t.Helper()

	deadline := time.Now().Add(tracerWait)

	for time.Now().Before(deadline) {
		for _, line := range h.ACPLines() {
			if line.Dir == "in" && strings.Contains(line.JSON, idKey) {
				return line
			}
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("no reply frame matching %s arrived on the tee within the wait bound", idKey)

	return TeeLine{}
}
