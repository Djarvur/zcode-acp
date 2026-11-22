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
	"encoding/json"
	"strings"
	"testing"

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
