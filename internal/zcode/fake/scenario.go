package fake

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// errInvalidScenario is the sentinel wrapped by every loader validation
// failure; the message names the offending field (fail-loud style).
var errInvalidScenario = errors.New("invalid scenario")

// Scenario is the declarative scripting surface (D-01): a request table
// with per-request replies and ordered side effects, an unknown-method
// reply, and the stdin-EOF drain window. Schema keys are snake_case (the
// repo's tagliatelle contract); wire payloads inside the raw fields keep
// the dialect's camelCase verbatim from the captures.
type Scenario struct {
	Name          string        `json:"name"`
	OnRequest     []RequestRule `json:"on_request"`
	UnknownMethod *Reply        `json:"unknown_method"`
	EOFDrainMs    int           `json:"eof_drain_ms"`
}

// RequestRule answers one request method: the reply envelope payload plus
// the ordered side effects to run around it. Times, when positive, caps how
// many requests the rule matches — once the budget is spent, a later
// on_request entry for the same method takes over (the -32010 double-send
// corpus row's second-send error rule).
type RequestRule struct {
	Method      string       `json:"method"`
	Times       int          `json:"times,omitempty"`
	DelayMs     int          `json:"delay_ms"`
	Reply       *Reply       `json:"reply"`
	SideEffects []SideEffect `json:"side_effects"`

	// matched counts rule matches at serve time (guarded by the server's
	// ruleMu — handler goroutines are concurrent); never serialized.
	matched int
}

// Reply is the reply payload of a request rule: exactly one of result or
// error, byte-exact from the capture-quoted shapes.
type Reply struct {
	Result json.RawMessage  `json:"result,omitempty"`
	Error  *zcode.WireError `json:"error,omitempty"`
}

// SideEffect is exactly one of emit (a notification) or reverse (a blocking
// server→host request).
type SideEffect struct {
	Emit    *Emit    `json:"emit,omitempty"`
	Reverse *Reverse `json:"reverse,omitempty"`
}

// Emit writes one notification (method, params, no id) to the host.
type Emit struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Reverse issues one server→host request and blocks until the host answers
// or timeout_ms elapses (recorded honestly as unanswered, never normalized
// away). Reannounce, when present, scripts the broker's re-announce
// mechanics (05-RESEARCH.md Pattern 2): the same method and params
// re-issued under NEW wire ids while the first envelope keeps the
// registered expect_answer.
type Reverse struct {
	Method       string          `json:"method"`
	Params       json.RawMessage `json:"params"`
	TimeoutMs    int             `json:"timeout_ms"`
	ExpectAnswer *Reply          `json:"expect_answer,omitempty"`
	Reannounce   *Reannounce     `json:"reannounce,omitempty"`
}

// Reannounce scripts a reverse request's re-announce schedule: the same
// method and params re-issued count times, spaced interval_ms apart, each
// under a NEW wire id (the server allocates a fresh id per send). The first
// envelope owns the declared expect_answer; re-announce envelopes carry no
// expectation of their own.
type Reannounce struct {
	Count      int `json:"count"`
	IntervalMs int `json:"interval_ms"`
}

// LoadScenario reads and validates a scenario JSON file. Decoding is strict
// (unknown schema keys are rejections, never silent defaults — a typo'd key
// would silently change the scripted wire behavior), and every raw payload is
// compacted at load so authored fixtures may be pretty-printed while the wire
// stays one line per envelope (dialect §1 framing).
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fake: load scenario: %w", err)
	}

	var sc Scenario

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&sc); err != nil {
		return nil, fmt.Errorf("fake: parse scenario %s: %w", path, err)
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: %s must hold exactly one JSON scenario document", errInvalidScenario, path)
	}

	if err := sc.validate(); err != nil {
		return nil, fmt.Errorf("fake: scenario %s: %w", path, err)
	}

	return &sc, nil
}

// validate enforces the scenario contract, naming the offending field.
func (s *Scenario) validate() error {
	if s.Name == "" {
		return fmt.Errorf("%w: name required", errInvalidScenario)
	}

	if s.EOFDrainMs < 0 {
		return fmt.Errorf("%w: eof_drain_ms must be >= 0", errInvalidScenario)
	}

	if s.UnknownMethod != nil {
		if err := s.UnknownMethod.validate("unknown_method"); err != nil {
			return err
		}
	}

	if len(s.OnRequest) == 0 {
		// A scenario must declare how it terminates: with no request rules,
		// the eof_drain_ms window is the only declared end (dialect §1 EOF
		// facts).
		if s.EOFDrainMs <= 0 {
			return fmt.Errorf("%w: scenario must declare how it terminates: "+
				"eof_drain_ms required when on_request is empty", errInvalidScenario)
		}

		return nil
	}

	for i := range s.OnRequest {
		if err := s.OnRequest[i].validate(i); err != nil {
			return err
		}
	}

	return nil
}

// validate checks one request rule (method, reply, ordered side effects).
func (r *RequestRule) validate(idx int) error {
	if r.Method == "" {
		return fmt.Errorf("%w: on_request[%d].method required", errInvalidScenario, idx)
	}

	if r.DelayMs < 0 {
		return fmt.Errorf("%w: on_request[%d].delay_ms must be >= 0", errInvalidScenario, idx)
	}

	if r.Times < 0 {
		return fmt.Errorf("%w: on_request[%d].times must be >= 0", errInvalidScenario, idx)
	}

	if r.Reply == nil {
		return fmt.Errorf("%w: on_request[%d].reply required", errInvalidScenario, idx)
	}

	where := fmt.Sprintf("on_request[%d].reply", idx)

	if err := r.Reply.validate(where); err != nil {
		return err
	}

	for si := range r.SideEffects {
		if err := r.SideEffects[si].validate(idx, si); err != nil {
			return err
		}
	}

	return nil
}

// validate checks that the reply carries exactly a usable payload and
// compacts it byte-deterministically. An error payload must name its dialect
// §6 numeric code — consumers match the code, never the message text.
func (r *Reply) validate(where string) error {
	if r.Result == nil && r.Error == nil {
		return fmt.Errorf("%w: %s needs result or error", errInvalidScenario, where)
	}

	if r.Result != nil {
		if err := compactRaw(&r.Result); err != nil {
			return fmt.Errorf("%w: %s.result: %w", errInvalidScenario, where, err)
		}
	}

	if r.Error != nil && r.Error.Code == 0 {
		return fmt.Errorf("%w: %s.error.code required (dialect §6 numeric code)", errInvalidScenario, where)
	}

	return nil
}

// validate checks one side effect: exactly one of emit or reverse, with the
// payloads compacted for the wire.
func (se *SideEffect) rule() string {
	switch {
	case se.Emit != nil:
		return "emit"

	case se.Reverse != nil:
		return "reverse"
	}

	return ""
}

// validate checks one side effect: exactly one of emit/reverse must be
// present — both is ambiguous, neither is inert.
func (se *SideEffect) validate(reqIdx, sideIdx int) error {
	where := fmt.Sprintf("on_request[%d].side_effects[%d]", reqIdx, sideIdx)

	if se.Emit != nil && se.Reverse != nil {
		return fmt.Errorf("%w: %s must carry exactly one of emit or reverse", errInvalidScenario, where)
	}

	kind := se.rule()
	if kind == "" {
		return fmt.Errorf("%w: %s needs emit or reverse", errInvalidScenario, where)
	}

	if se.Emit != nil {
		return se.Emit.validate(where)
	}

	return se.Reverse.validate(where)
}

// validate checks one emit side effect.
func (e *Emit) validate(where string) error {
	if e.Method == "" {
		return fmt.Errorf("%w: %s.emit.method required", errInvalidScenario, where)
	}

	if len(e.Params) == 0 {
		return fmt.Errorf("%w: %s.emit.params required", errInvalidScenario, where)
	}

	if err := compactRaw(&e.Params); err != nil {
		return fmt.Errorf("%w: %s.emit.params: %w", errInvalidScenario, where, err)
	}

	return nil
}

// validate checks one reverse side effect (timeout, compact payloads).
func (r *Reverse) validate(where string) error {
	if r.Method == "" {
		return fmt.Errorf("%w: %s.reverse.method required", errInvalidScenario, where)
	}

	if len(r.Params) == 0 {
		return fmt.Errorf("%w: %s.reverse.params required", errInvalidScenario, where)
	}

	if r.TimeoutMs <= 0 {
		return fmt.Errorf("%w: %s.reverse.timeout_ms must be > 0", errInvalidScenario, where)
	}

	if err := compactRaw(&r.Params); err != nil {
		return fmt.Errorf("%w: %s.reverse.params: %w", errInvalidScenario, where, err)
	}

	if r.ExpectAnswer != nil {
		if err := r.ExpectAnswer.validate(where + ".expect_answer"); err != nil {
			return err
		}
	}

	if r.Reannounce != nil {
		if err := r.Reannounce.validate(where + ".reannounce"); err != nil {
			return err
		}
	}

	return nil
}

// validate checks one re-announce schedule: a positive count (a zero count
// scripts nothing) and a non-negative interval (a negative delay is
// meaningless) — fail loud naming the field.
func (ra *Reannounce) validate(where string) error {
	if ra.Count <= 0 {
		return fmt.Errorf("%w: %s.count must be > 0", errInvalidScenario, where)
	}

	if ra.IntervalMs < 0 {
		return fmt.Errorf("%w: %s.interval_ms must be >= 0", errInvalidScenario, where)
	}

	return nil
}

// compactRaw compacts a raw JSON payload in place so authored whitespace
// never leaks into wire lines or goldens.
func compactRaw(raw *json.RawMessage) error {
	var buf bytes.Buffer
	if err := json.Compact(&buf, *raw); err != nil {
		return fmt.Errorf("compact payload: %w", err)
	}

	*raw = json.RawMessage(buf.Bytes())

	return nil
}
