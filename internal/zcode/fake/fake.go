// Package fake provides a scriptable in-process ZCode app-server (D-01):
// Serve speaks the same newline-delimited envelope grammar as the codec over
// any io.ReadWriteCloser (io.Pipe in-process, exec pipes for the standalone
// binary), replays a declarative Scenario, and records a direction-tagged
// Transcript that golden tests replay byte-deterministically.
package fake

import (
	"errors"
	"io"
)

// errSkeleton marks the scripting core that lands with the conn/fake GREEN
// commit (plan Task 3). The tracer RED fails on this error.
var errSkeleton = errors.New("skeleton: not implemented")

// Dir tags which way a transcript frame travelled relative to the fake.
type Dir string

const (
	// DirIn marks a host→fake frame: requests and reverse-request replies.
	DirIn Dir = "in"

	// DirOut marks a fake→host frame: replies, notifications, and the
	// fake's own reverse requests.
	DirOut Dir = "out"

	// DirNote marks a non-frame transcript annotation — e.g. a reverse
	// request recorded honestly as unanswered at its timeout (record-
	// honesty discipline: never normalize a missing frame away).
	DirNote Dir = "note"
)

// Line is one transcript entry: the direction tag plus the raw wire line.
type Line struct {
	Dir  Dir
	JSON string
}

// Transcript is the recorded wire conversation in true wire order — the
// protocol-spec-made-executable fixture (D-01). Fixed scenario ids keep
// replays byte-deterministic.
type Transcript struct {
	Lines []Line
}

// Serve runs the scripted app-server over rw against sc until stdin EOF,
// then drains sc.EOFDrainMs and returns the recorded transcript. Requests
// are matched against sc.on_request, replies honor delay_ms, side effects
// run in order (emit notifications, issue blocking reverse requests), and
// unknown methods get sc.unknown_method.
//
// Skeleton stage: the scripting loop lands with the conn/fake commit (plan
// Task 3); until then Serve reports errSkeleton so the tracer RED fails on
// its own assertion.
func Serve(rw io.ReadWriteCloser, sc *Scenario) (Transcript, error) {
	_, _ = rw, sc

	return Transcript{}, errSkeleton
}
