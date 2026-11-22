package turn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Watch argument-validation sentinels (fail-loud construction contract).
var (
	errWatcherSessionID = errors.New("turn: poll watcher: sessionID required")
	errWatcherOnEvent   = errors.New("turn: poll watcher: onEvent required")
)

// Quiet-window gate defaults (dialect §5 :141 and the probe's
// QUIET_SEND_FLOOR_MULTIPLE = 3, experiments/app-server-probe.mjs:604):
// quiet 2.5 s, floor = 3x quiet measured from the send ack, polls every
// 500 ms. Tests inject milliseconds so no test waits on these.
const (
	DefaultQuietMs        = 2500
	DefaultFloorMultiple  = 3
	DefaultPollIntervalMs = 500
)

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

// ActivityFeed is the optional TurnWatcher extension for watchers whose
// settle gate consumes live wire activity (QA-03). The machine — the ONE
// observation funnel (T-3-06) — feeds every mid-turn envelope to a watcher
// implementing it; no separate conn tap is registered (a per-Watch
// registration would race the teardown-baseline observer counts and split
// the one-observe-path discipline).
type ActivityFeed interface {
	// ObserveActivity receives one envelope observed mid-turn. It runs on
	// the observing goroutine (the conn read goroutine on the real path)
	// and must not block.
	ObserveActivity(env zcode.Envelope)
}

// PollWatcher is the polling TurnWatcher implementation: it polls
// session/messages and gates the quiet-window fallback exactly like the
// probe harness (experiments/app-server-probe.mjs:555-601 — the fallback
// completion path, dialect §5 :141): a send-latency floor of FloorMultiple
// x QuietMs measured from the send ack, then the settle gate (QuietMs with
// no new message activity). Both live-verified gates hold the quiet fire
// until the turn can actually be over: the floor keeps first-token request
// latency from masquerading as silence, and the settle gate keeps streaming
// activity from ending the turn mid-generation.
//
// Activity is ANY live wire traffic for the watched session (QA-03): the
// machine feeds every mid-turn envelope through ObserveActivity, and every
// session-scoped one — the session/event delta stream, the v4 telemetry
// chatter, the operation events — resets the settle gate. The poll surface
// alone cannot see a turn that is still generating (the reply text rides
// the notification stream, not the messages projection), which is exactly
// how the 2026-11-22 field runs ended turns mid-generation.
type PollWatcher struct {
	conn   *zcode.Conn
	source PollSource

	quietMs        int
	floorMultiple  int
	pollIntervalMs int

	// actMu guards the settle-gate activity state shared between the loop
	// goroutine and the machine's ObserveActivity feed; watchedSession is
	// the live Watch's session (set at Watch, cleared at stop).
	actMu          sync.Mutex
	watchedSession string
	lastActivity   time.Time
	lastRaw        json.RawMessage
}

// NewPollWatcher returns the polling watcher for conn with cfg applied
// (zero knobs take the dialect defaults; a nil Source polls through conn).
func NewPollWatcher(conn *zcode.Conn, cfg PollConfig) *PollWatcher {
	w := &PollWatcher{
		conn:           conn,
		source:         cfg.Source,
		quietMs:        cfg.QuietMs,
		floorMultiple:  cfg.FloorMultiple,
		pollIntervalMs: cfg.PollIntervalMs,
	}

	if w.source == nil {
		w.source = connPollSource{conn: conn}
	}

	if w.quietMs <= 0 {
		w.quietMs = DefaultQuietMs
	}

	if w.floorMultiple <= 0 {
		w.floorMultiple = DefaultFloorMultiple
	}

	if w.pollIntervalMs <= 0 {
		w.pollIntervalMs = DefaultPollIntervalMs
	}

	return w
}

// Watch implements TurnWatcher: the poll loop runs on its own goroutine and
// stop joins it before returning, so no delivery can happen after stop.
func (w *PollWatcher) Watch(ctx context.Context, sessionID string, onEvent func(Observation)) (stop func(), err error) {
	if sessionID == "" {
		return nil, errWatcherSessionID
	}

	if onEvent == nil {
		return nil, errWatcherOnEvent
	}

	watchCtx, cancel := context.WithCancel(ctx)
	exited := make(chan struct{})

	w.actMu.Lock()
	w.watchedSession = sessionID
	w.actMu.Unlock()

	go func() {
		defer close(exited)

		w.loop(watchCtx, sessionID, onEvent)
	}()

	return func() {
		cancel()

		w.actMu.Lock()
		w.watchedSession = ""
		w.actMu.Unlock()

		<-exited
	}, nil
}

// ObserveActivity implements ActivityFeed: a notification scoped to the
// watched session is evidence of life and resets the settle gate.
// Host-level traffic carries no sessionId (process telemetry, startup
// pushes) and foreign-session envelopes are other conversations — neither
// is this turn's reply stream, so neither counts. The watcher's own poll
// echo (the synthetic session/messages envelope) is presence evidence,
// not wire life.
func (w *PollWatcher) ObserveActivity(env zcode.Envelope) {
	if env.Method == zcode.MethodSessionMessages {
		return
	}

	w.actMu.Lock()
	defer w.actMu.Unlock()

	if w.watchedSession == "" {
		return
	}

	if sid := paramsString(env.Params, "sessionId"); sid != w.watchedSession {
		return
	}

	w.lastActivity = time.Now()
}

// loop is the watcher body: poll on the injected cadence, deliver observed
// envelopes to onEvent, reset the settle gate whenever ANY watched-session
// activity lands (poll-surface change or notification-stream envelope), and
// fire the quiet-window observation once BOTH gates pass — the send-latency
// floor (FloorMultiple x QuietMs from the watcher start, which the machine
// places right after the send ack) and then QuietMs with no new activity.
// The settle does not end the loop: it re-arms (a full fresh quiet window)
// and keeps delivering, so the machine's start-announced veto (QA-03) can
// hold the turn open for the promised terminal signal — the non-veto path
// resolves on the first settle and the machine's stop joins the loop.
// Gate granularity is the poll cadence: session/messages carries
// presence/time evidence only (RESEARCH Open Question 2 resolution).
func (w *PollWatcher) loop(ctx context.Context, sessionID string, onEvent func(Observation)) {
	quiet := time.Duration(w.quietMs) * time.Millisecond
	interval := time.Duration(w.pollIntervalMs) * time.Millisecond

	start := time.Now()
	floorAt := start.Add(time.Duration(w.floorMultiple) * quiet)

	w.actMu.Lock()
	w.lastActivity = start
	w.actMu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		envs, pollErr := w.source.Poll(ctx, sessionID)
		if pollErr != nil {
			if ctx.Err() != nil {
				return
			}

			// One failed poll (a wire hiccup, a refusal) is recorded, never
			// fatal: the next cadence tick retries, and a dead conn surfaces
			// through the machine's own Call paths.
			slog.Warn("turn: poll watcher poll failed", "session", sessionID, "err", pollErr)
		} else {
			for i := range envs {
				w.actMu.Lock()

				if !bytes.Equal(w.lastRaw, envs[i].Params) {
					w.lastActivity = time.Now()
					w.lastRaw = envs[i].Params
				}

				w.actMu.Unlock()

				onEvent(Observation{Env: &envs[i]})
			}
		}

		w.actMu.Lock()
		now := time.Now()
		lastActivity := w.lastActivity
		w.actMu.Unlock()

		if !now.Before(floorAt) && now.Sub(lastActivity) >= quiet {
			onEvent(Observation{Quiet: true})

			// Re-arm: the next settle needs a full fresh quiet window. The
			// start-announced veto consumes settles one window at a time
			// while the promised terminal signal is pending; a turn the
			// machine already resolved is simply joined by its stop.
			w.actMu.Lock()
			w.lastActivity = time.Now()
			w.actMu.Unlock()
		}
	}
}

// connPollSource is the real PollSource: one session/messages poll through
// the shared Conn per call, the result wrapped opaquely as ONE
// session-scoped observation envelope — presence/time evidence only
// (RESEARCH Open Question 2 resolution), content never interpreted.
type connPollSource struct {
	conn *zcode.Conn
}

// Poll implements PollSource.
func (s connPollSource) Poll(ctx context.Context, sessionID string) ([]zcode.Envelope, error) {
	res, err := s.conn.Call(ctx, zcode.MethodSessionMessages, map[string]any{"sessionId": sessionID})
	if err != nil {
		return nil, fmt.Errorf("turn: poll session/messages: %w", err)
	}

	return []zcode.Envelope{{
		Method: zcode.MethodSessionMessages,
		Params: res,
	}}, nil
}
