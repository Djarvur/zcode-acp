package turn

import (
	"context"
	"errors"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// errNotImplemented marks the RED stub bodies: the contract surface is
// declared, the behavior lands with the GREEN commit (03-05 Task 2).
var errNotImplemented = errors.New("turn: not implemented (RED stub)")

// Observation is one TurnWatcher delivery: a session-scoped envelope
// observed on the watched session, or the watcher's quiet-window
// settlement. Env is non-nil exactly when Quiet is false.
type Observation struct {
	Env   *zcode.Envelope // the observed session-scoped envelope
	Quiet bool            // true when the quiet-window settle gate fired
}

// TurnWatcher watches one app-server session and delivers turn signals
// until stopped (D-02 strategy seam). The polling implementation ships in
// Phase 3; the subscribe-based watcher is a reserved strategy slot, NOT
// implemented (CONTEXT Deferred Ideas).
type TurnWatcher interface {
	// Watch delivers observations for sessionID to onEvent until stop is
	// called or ctx is cancelled. stop is synchronous: after it returns no
	// further delivery happens. onEvent must not block (it runs on the
	// watcher goroutine).
	Watch(ctx context.Context, sessionID string, onEvent func(Observation)) (stop func(), err error)
}

// PollSource is the PollWatcher's seam over the session/messages surface.
// One poll returns the session-scoped observation envelopes to deliver; the
// real source wraps the poll result opaquely (RESEARCH Open Question 2
// resolution: messages are presence/time evidence only — content is never
// interpreted, no delta cursors). Tests stub the source.
type PollSource interface {
	Poll(ctx context.Context, sessionID string) ([]zcode.Envelope, error)
}

// PollConfig is the PollWatcher's construction config: the quiet-window
// gate knobs (dialect §5 :141; the probe's QUIET_SEND_FLOOR_MULTIPLE) and
// the injectable poll source. Zero knobs take the dialect defaults; tests
// inject milliseconds so no test waits on wall-clock defaults.
type PollConfig struct {
	// Source, when nil, polls session/messages through the watcher's Conn.
	Source PollSource

	// QuietMs is the settle-gate window: quiet milliseconds with no new
	// message activity. Default DefaultQuietMs (2500).
	QuietMs int

	// FloorMultiple is the send-latency floor multiple: the settle gate may
	// not fire before FloorMultiple x QuietMs after the send ack. Default
	// DefaultFloorMultiple (3).
	FloorMultiple int

	// PollIntervalMs is the session/messages poll cadence. Default
	// DefaultPollIntervalMs (500).
	PollIntervalMs int
}

// PollWatcher is the polling TurnWatcher implementation: it polls
// session/messages and gates the quiet-window fallback exactly like the
// probe harness (experiments/app-server-probe.mjs:555-601 — the only
// end-to-end-proven completion path, dialect §5 :141): a send-latency floor
// of FloorMultiple x QuietMs measured from the send ack, then the settle
// gate (QuietMs with no new message activity).
type PollWatcher struct {
	conn   *zcode.Conn
	source PollSource

	quietMs        int
	floorMultiple  int
	pollIntervalMs int
}

// NewPollWatcher returns the polling watcher for conn with cfg applied
// (zero knobs take the dialect defaults; a nil Source polls through conn).
// RED stub: config is accepted, behavior lands with the GREEN commit.
func NewPollWatcher(conn *zcode.Conn, cfg PollConfig) *PollWatcher {
	_, _ = conn, cfg

	return &PollWatcher{}
}

// Watch implements TurnWatcher. RED stub: returns errNotImplemented.
func (w *PollWatcher) Watch(ctx context.Context, sessionID string, onEvent func(Observation)) (stop func(), err error) {
	_, _, _, _ = w, ctx, sessionID, onEvent

	return nil, errNotImplemented
}
