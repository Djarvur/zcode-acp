package acpfront_test

import (
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"
)

// setModelScenario is the corpus row the front-tier contract locks ride
// (plan 04-02): its create/subscribe/setModel preamble IS the D-03 wire
// contract. The initialize lock needs no back-leg traffic — the row is
// merely the composed harness's scenario.
const setModelScenario = "setmodel-followup"

// TestInitializeHonestCapabilities — the CORE-01 honesty lock (plan 04-03,
// D-02): the initialize RESPONSE is asserted on the RAW recorded tee frame,
// decoded NOTHING — the wire truth is the contract, not the decoded struct.
// The executed-probe shape (04-RESEARCH "Code Examples") is the golden
// reference: protocolVersion 1, real agentInfo, exactly the two advertised
// auth methods in contract order (agent variant, no type field), zero-value
// capability objects — and NO loadSession/fs/terminal surface anywhere
// (absence assertions, RESEARCH A7: assert absent features, not
// empty-object cosmetics). This lock rides the plan 04-02 harness as
// composed by plan 04-06 (newTracerFront over the setmodel-followup corpus).
func TestInitializeHonestCapabilities(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	lines := h.ACPLines()
	initIdx := teeLineIndex(t, lines, "in", `"protocolVersion"`, "initialize response")
	initRaw := lines[initIdx].JSON

	// The EXACT executed-probe honest shape, byte for byte, WIDENED by plan
	// 06-04 (MODEL-01, D-11): sessionCapabilities gains the options
	// empty-struct opt-in. One lock covers protocolVersion 1, the agentInfo
	// identity, and both authMethods in contract order without a type field
	// (the agent variant is the default).
	probeShape := `{"agentCapabilities":{"auth":{},"mcpCapabilities":{},"promptCapabilities":{},` +
		`"sessionCapabilities":{"options":{}}},` +
		`"agentInfo":{"name":"zcode-acp","title":"zcode-acp agent","version":"0.0.0-dev"},` +
		`"authMethods":[{"id":"zai-api-key","name":"z.ai API key"},` +
		`{"id":"zai-individual-coding-plan","name":"z.ai Individual Coding Plan"}],"protocolVersion":1}`

	if !strings.Contains(initRaw, `"result":`+probeShape) {
		t.Fatalf("initialize response is not the executed-probe honest shape:\n got: %s\nwant result: %s",
			initRaw, probeShape)
	}

	// Absence assertions (D-02/A7): the features this build does not offer
	// must not appear anywhere on the wire — capability omission is
	// spec-correct and the initialize frame must gain no fields this phase.
	for _, banned := range []string{"loadSession", "readTextFile", "writeTextFile", "terminal", `"fs"`} {
		if strings.Contains(initRaw, banned) {
			t.Fatalf("initialize response advertises %s (D-02 honesty violation): %s", banned, initRaw)
		}
	}
}
