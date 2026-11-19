package fake_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Fail-loud loader validation (03-03 Task 1): every invalid scenario is
// rejected with an error naming the offending field — never silently
// defaulted. Style precedent: scrub-capture IN-04 (invalid input exits
// non-zero with a message naming the contract). Invalid fixtures are written
// inline to the test's temp dir — they never live under testdata/, where
// they would pollute the golden corpus the replay test iterates.

// writeInvalidScenario writes one inline invalid scenario JSON to the test's
// temp dir and returns its path.
func writeInvalidScenario(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "invalid-scenario.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write invalid scenario: %v", err)
	}

	return path
}

// assertScenarioRejected loads the scenario and requires a rejection whose
// message names the offending field: every want substring must appear in the
// error. A scenario that loads without error is the exact fail-open this
// contract forbids.
func assertScenarioRejected(t *testing.T, path string, want ...string) {
	t.Helper()

	_, err := fake.LoadScenario(path)
	if err == nil {
		t.Fatalf("invalid scenario loaded without error (fail-open): %s", path)
	}

	for _, sub := range want {
		if !strings.Contains(err.Error(), sub) {
			t.Fatalf("rejection must name the offending field %q, got: %v", sub, err)
		}
	}
}

// TestLoadScenarioRejectsInvalidMissingName: a scenario without a name is
// rejected, naming the name field.
func TestLoadScenarioRejectsInvalidMissingName(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "on_request": [{"method": "session/create", "reply": {"result": {}}}],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "name")
}

// TestLoadScenarioRejectsInvalidRequestRuleWithoutMethod: an on_request
// entry without a method is rejected, naming the entry's method field.
func TestLoadScenarioRejectsInvalidRequestRuleWithoutMethod(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "no-method",
	  "on_request": [{"reply": {"result": {}}}],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "on_request[0].method")
}

// TestLoadScenarioRejectsInvalidRequestRuleWithoutReplyAndSideEffects: an
// on_request entry carrying neither a reply nor side effects cannot produce
// any wire behavior — rejected, naming the reply field.
func TestLoadScenarioRejectsInvalidRequestRuleWithoutReplyAndSideEffects(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "no-reply",
	  "on_request": [{"method": "session/create"}],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "on_request[0].reply")
}

// TestLoadScenarioRejectsInvalidSideEffectUnknownKey: a side effect carrying
// an unknown key (a typo'd schema field) is rejected naming the key — a
// silent ignore would bury the typo and change the scripted wire behavior.
func TestLoadScenarioRejectsInvalidSideEffectUnknownKey(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "unknown-key",
	  "on_request": [{
	    "method": "session/create",
	    "reply": {"result": {}},
	    "side_effects": [{
	      "emit": {"method": "state.updated", "params": {"patch": {"status": "running"}}},
	      "emits": {"method": "state.updated", "params": {"patch": {"status": "running"}}}
	    }]
	  }],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "emits")
}

// TestLoadScenarioRejectsInvalidSideEffectBothKinds: a side effect carrying
// BOTH emit and reverse is ambiguous — exactly one is allowed, and the
// rejection names the constraint.
func TestLoadScenarioRejectsInvalidSideEffectBothKinds(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "both-kinds",
	  "on_request": [{
	    "method": "session/create",
	    "reply": {"result": {}},
	    "side_effects": [{
	      "emit": {"method": "state.updated", "params": {"patch": {"status": "running"}}},
	      "reverse": {"method": "session/requestRuntimePreferences", "params": {"sessionId": "sess_fix1"}, "timeout_ms": 100}
	    }]
	  }],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "exactly one")
}

// TestLoadScenarioRejectsInvalidReverseWithoutTimeout: a reverse side effect
// without timeout_ms can never resolve (it would block forever) — rejected,
// naming the timeout_ms field.
func TestLoadScenarioRejectsInvalidReverseWithoutTimeout(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "no-reverse-timeout",
	  "on_request": [{
	    "method": "session/create",
	    "reply": {"result": {}},
	    "side_effects": [{
	      "reverse": {"method": "session/requestRuntimePreferences", "params": {"sessionId": "sess_fix1"}}
	    }]
	  }],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "timeout_ms")
}

// TestLoadScenarioRejectsInvalidReverseWithoutMethod: a reverse side effect
// without a method names the reverse's method field.
func TestLoadScenarioRejectsInvalidReverseWithoutMethod(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "no-reverse-method",
	  "on_request": [{
	    "method": "session/create",
	    "reply": {"result": {}},
	    "side_effects": [{"reverse": {"params": {"sessionId": "sess_fix1"}, "timeout_ms": 100}}]
	  }],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "reverse.method")
}

// TestLoadScenarioRejectsInvalidNegativeDelay: a negative delay_ms is
// rejected, naming the delay_ms field.
func TestLoadScenarioRejectsInvalidNegativeDelay(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "negative-delay",
	  "on_request": [{"method": "session/create", "delay_ms": -5, "reply": {"result": {}}}],
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "delay_ms")
}

// TestLoadScenarioRejectsInvalidUnknownMethodWithoutCode: an unknown_method
// error reply without its dialect §6 numeric code is rejected — error
// payloads must name their code (consumers match the numeric code, never the
// message text).
func TestLoadScenarioRejectsInvalidUnknownMethodWithoutCode(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "no-error-code",
	  "on_request": [{"method": "session/create", "reply": {"result": {}}}],
	  "unknown_method": {"error": {"message": "MethodNotFound"}},
	  "eof_drain_ms": 100
	}`)

	assertScenarioRejected(t, path, "unknown_method.error.code")
}

// TestLoadScenarioRejectsInvalidNegativeEofDrain: a negative eof_drain_ms is
// rejected, naming the eof_drain_ms field.
func TestLoadScenarioRejectsInvalidNegativeEofDrain(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "negative-drain",
	  "on_request": [{"method": "session/create", "reply": {"result": {}}}],
	  "eof_drain_ms": -1
	}`)

	assertScenarioRejected(t, path, "eof_drain_ms")
}

// TestLoadScenarioRejectsInvalidEmptyRequestTableWithoutTermination: an
// empty on_request table combined with a missing eof_drain_ms declares
// neither wire behavior nor how the scenario terminates — rejected, naming
// the termination contract.
func TestLoadScenarioRejectsInvalidEmptyRequestTableWithoutTermination(t *testing.T) {
	path := writeInvalidScenario(t, `{
	  "name": "no-termination",
	  "on_request": []
	}`)

	assertScenarioRejected(t, path, "eof_drain_ms", "terminates")
}
