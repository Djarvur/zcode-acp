package acpfront

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// Per-turn context ownership (RESEARCH Pattern 5, CORE-05): each prompt
// turn runs on a FRONT-OWNED context derived from the Front's base
// context — never the SDK request context. The SDK maps a handler's
// context.Canceled to a -32800 RequestCancelled ERROR response, while ACP
// requires session/cancel answered with stopReason:"cancelled" as a
// SUCCESS result — so the turn context is cancelled only when the prompt
// returns (release), and cancellation reaches the turn exclusively through
// machine.Cancel (idempotent, exactly once: Phase 3), which resolves
// StopCancelled via the observed stop outcome. Prompt's cancel fan-in
// goroutine selects {SDK ctx.Done → machine.Cancel; turn ctx.Done →
// return} so both cancel sources trigger exactly one machine.Cancel, and
// post-cancel delta drain still reaches the client because the tap writes
// on the turn context (alive until the prompt returns).
//
// WR-03 (plan 05-04, the OQ3 disposition): the tap-facing emitter is the
// per-turn pump's non-blocking enqueue — the backend read goroutine never
// parks on the ACP wire. The pump is a byte-budgeted queue drained to the
// sink by its own goroutine at wire pace: same-toolCallId tool update
// patches coalesce keep-latest (the last status wins — patch semantics per
// spec), text/thought deltas stay lossless, and hitting the byte ceiling
// FAILS THE TURN LOUD (a coded prompt error naming the cause — never a
// hang, never a silent drop). Release flushes before Prompt returns: the
// slot clears, the turn context cancels, the pump's final drain delivers
// whatever was legally queued (bounded — a sink parked past the bound
// abandons the drain so Prompt can never hang), and no note is accepted
// after the slot clears.

// defaultStreamCeiling is the documented per-turn queue ceiling (the OQ3
// disposition: a per-turn MiB-scale memory ceiling — 64 MiB).
const defaultStreamCeiling = 64 << 20

// pumpReleaseWait bounds release's drain wait — the bounded post-return
// write window: past the bound the pump is abandoned (drains-or-discards)
// so Prompt returns and the stopReason rides after the best-effort flush.
const pumpReleaseWait = 2 * time.Second

// pumpEntry is one queued note with its byte-accounting weight.
type pumpEntry struct {
	note acpport.SessionNote
	size int
}

// turnPump is the WR-03 per-turn pump: the tap's enqueue never blocks, a
// dedicated goroutine drains to the emitter closure at sink pace.
type turnPump struct {
	emit    func(acpport.SessionNote) error
	cancel  context.CancelFunc // the turn ctx cancel — a ceiling breach unwinds the turn loud
	ceiling int

	mu     sync.Mutex
	queue  []pumpEntry
	bytes  int
	breach error // non-nil → the ceiling was hit; every later enqueue is refused
	notify chan struct{}
	done   chan struct{}
}

// newTurnPump builds the pump for one turn and starts its drain goroutine
// on the turn context.
func newTurnPump(
	turnCtx context.Context,
	emit func(acpport.SessionNote) error,
	cancel context.CancelFunc,
	ceiling int,
) *turnPump {
	p := &turnPump{
		emit:    emit,
		cancel:  cancel,
		ceiling: ceiling,
		notify:  make(chan struct{}, 1),
		done:    make(chan struct{}),
	}

	go p.run(turnCtx)

	return p
}

// run drains the queue until the turn context dies (release or breach),
// then delivers whatever was legally queued (the final flush) and exits.
func (p *turnPump) run(turnCtx context.Context) {
	defer close(p.done)

	for {
		select {
		case <-turnCtx.Done():
			p.drain()

			return
		case <-p.notify:
			p.drain()
		}
	}
}

// drain pops and delivers the queue to the emitter closure, one note at a
// time. A failed write (a stalled or closing ACP wire) is recorded
// structurally — never with params/content — and the drain continues; the
// prompt answer carries the honest outcome.
func (p *turnPump) drain() {
	for {
		p.mu.Lock()

		if len(p.queue) == 0 {
			p.mu.Unlock()

			return
		}

		entry := p.queue[0]
		p.queue = p.queue[1:]
		p.mu.Unlock()

		if err := p.emit(entry.note); err != nil {
			slog.Warn("acpfront: session update write failed", "err", err)
		}
	}
}

// enqueue accepts one note onto the queue — non-blocking by construction
// (append under the mutex; the drain goroutine owns the wire). A ceiling
// breach refuses the note and fails the turn loud: the recorded breach
// error is what Prompt surfaces (a coded prompt error naming the cause),
// and the turn context cancel unwinds the machine promptly.
func (p *turnPump) enqueue(note acpport.SessionNote) error {
	size := noteBytes(note)

	p.mu.Lock()

	if p.breach != nil {
		err := p.breach
		p.mu.Unlock()

		return err
	}

	if p.bytes+size > p.ceiling {
		if p.breach == nil {
			p.breach = acpport.NewInternal(
				"acpfront: per-turn stream ceiling exceeded (%d > %d bytes) — "+
					"failing the turn loud rather than dropping streamed output",
				p.bytes+size, p.ceiling)
		}

		err := p.breach
		p.mu.Unlock()

		p.cancel() // fail loud: unwind the turn; Prompt surfaces the breach

		return err
	}

	// Keep-latest coalescing — ONLY for tool update notes sharing a
	// toolCallId (patch semantics per spec: the last status wins). The
	// coalesced note keeps its predecessor's queue position, so every
	// other frame's relative order is untouched. Text/thought deltas and
	// start cards are lossless — they append, and their pressure counts
	// toward the ceiling so overflow fails loud rather than dropping.
	if !isToolUpdate(note) {
		p.appendLocked(note, size)
		p.mu.Unlock()
		p.signal()

		return nil
	}

	for i := len(p.queue) - 1; i >= 0; i-- {
		queued := p.queue[i]

		if !isToolUpdate(queued.note) || queued.note.Tool.ToolCallID != note.Tool.ToolCallID {
			continue
		}

		p.bytes -= queued.size
		p.queue[i] = pumpEntry{note: note, size: size}
		p.bytes += size
		p.mu.Unlock()
		p.signal()

		return nil
	}

	p.appendLocked(note, size)
	p.mu.Unlock()
	p.signal()

	return nil
}

// appendLocked queues one note and adds its weight; the caller holds mu.
func (p *turnPump) appendLocked(note acpport.SessionNote, size int) {
	p.queue = append(p.queue, pumpEntry{note: note, size: size})
	p.bytes += size
}

// isToolUpdate reports whether the note is a tool_call_update patch (the
// coalescing class). Start cards are never coalesced — the client needs
// every card it was promised.
func isToolUpdate(note acpport.SessionNote) bool {
	return note.Kind == acpport.NoteTool && note.Tool != nil && !note.Tool.IsStart
}

// signal pokes the drain goroutine (never blocks — the channel is
// buffered to one and the drain re-checks until the queue is empty).
func (p *turnPump) signal() {
	select {
	case p.notify <- struct{}{}:
	default:
	}
}

// breachErr returns the recorded ceiling breach error, if any.
func (p *turnPump) breachErr() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.breach
}

// wait waits bounded for the drain goroutine to exit (the release flush).
func (p *turnPump) wait(bound time.Duration) {
	select {
	case <-p.done:
	case <-time.After(bound):
		// Bounded abandon (drains-or-discards): the pump is parked inside a
		// sink write that never returns; Prompt must not hang on it. A
		// later unblock delivers at most the abandoned notes — the turn's
		// answer is already gone.
		slog.Warn("acpfront: pump release wait elapsed; abandoning the drain", "bound", bound)
	}
}

// noteBytes weighs one note for the ceiling accounting — the payload
// bytes the note will push onto the wire, plus a fixed per-frame
// allowance. The ceiling bounds memory pressure, not exact wire bytes.
func noteBytes(note acpport.SessionNote) int {
	const perFrameAllowance = 128

	switch note.Kind {
	case acpport.NoteMessage, acpport.NoteThought:
		return len(note.Delta) + perFrameAllowance
	case acpport.NoteTool:
		t := note.Tool
		if t == nil {
			return perFrameAllowance
		}

		size := len(t.Title) + len(t.Name) + len(t.Kind) + len(t.Status) +
			len(t.RawInput) + len(t.RawOutput) + perFrameAllowance

		if t.Diff != nil {
			size += len(t.Diff.Path) + len(t.Diff.OldText) + len(t.Diff.NewText)
		}

		for _, loc := range t.Locations {
			size += len(loc.Path)
		}

		return size
	default:
		return perFrameAllowance
	}
}

// acquireTurn takes the front-wide turn slot for one prompt turn (the
// single-active-turn invariant). It returns the front-owned turn context
// and the slot release func; release is nil when the slot is already held
// — a second prompt while a turn is active, the D-05 guard surface (the
// exact error class is plan 04-05's RED target). A non-nil error means the
// session is mid-teardown (SESS-01 closing gate): the caller surfaces the
// pre-validated ErrNoSession-class rejection — a prompt never acquires the
// slot while close tears its session down (T-5-12). The tap-facing emitter
// is the WR-03 pump's enqueue; the pump publishes under the same lock, so
// the stream.go tap observes either the complete turn state or none.
func (f *Front) acquireTurn(sessionID string, sink acpport.UpdateSink) (context.Context, func(), error) {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	// The SESS-01 closing gate under the slot lock: once close marks the
	// session, no mid-teardown acquisition can slip past the prompt's own
	// pre-check. (lifecycle.go — the T-5-12 race gate.)
	if f.isClosing(sessionID) {
		return nil, nil, acpport.NewInvalidRequest(
			"session/prompt: %s: the session is closing or closed", sessionID)
	}

	if f.turnActive {
		return nil, nil, nil
	}

	// Front-owned: derived from the front's base context, NOT the SDK
	// request context (the type doc — cancelling on the SDK ctx would turn
	// the cancelled stopReason into a -32800 error response).
	turnCtx, turnCancel := f.newTurnCtx()

	ceiling := f.streamCeiling
	if ceiling <= 0 {
		ceiling = defaultStreamCeiling
	}

	// The WR-03 pump: the emitter closure (sink + session id + turn ctx)
	// drains at wire pace on the pump goroutine; the tap's enqueue never
	// blocks the backend read loop.
	pump := newTurnPump(turnCtx, func(note acpport.SessionNote) error {
		note.SessionID = sessionID

		return sink(turnCtx, note)
	}, turnCancel, ceiling)

	// The SESS-01 close teardown's wait surface: done closes when release
	// has fully unwound the turn (flush included), so close's bounded
	// cancel-then-drain proceeds only after the turn is off the wire.
	done := make(chan struct{})

	f.turnActive = true
	f.turnSession = sessionID
	f.turnDone = done
	f.turnEmit = pump.enqueue
	f.turnPump = pump

	// The tap's turn-scoped classification state (plan 05-04): the slot's
	// turnId starts empty (record-on-first-sight at the tap, IN-05), the
	// cwd anchors the absolute diff/location paths (Pitfall 5), and the
	// diff registry starts fresh per turn.
	f.turnID = ""
	f.turnCwd = f.sessionCwdLocked(sessionID)
	f.turnDiffs = make(map[string]acpport.ToolDiff)

	release := func() {
		f.turnMu.Lock()

		f.turnActive = false
		f.turnSession = ""
		f.turnDone = nil
		f.turnEmit = nil
		f.turnID = ""
		f.turnCwd = ""
		f.turnDiffs = nil
		pump := f.turnPump
		f.turnPump = nil

		f.turnMu.Unlock()

		// Outside the lock. Cancel first: it wakes the pump's select (and
		// the fan-in goroutine's turnCtx.Done arm); the pump's final drain
		// delivers whatever was legally queued — the flush-before-stopReason
		// property lives HERE, inside release, before Prompt's stopReason is
		// returned. The wait is bounded: a sink parked past the bound
		// abandons the drain so Prompt can never hang on the ACP wire.
		turnCancel()
		pump.wait(pumpReleaseWait)

		// The teardown's drain signal — after the flush, never before, so
		// close's backend session/close cannot outrun the turn's last write.
		close(done)
	}

	return turnCtx, release, nil
}

// sessionCwdLocked reads the session-cwd registry; the caller holds turnMu
// (a different mutex than the registry's cwdMu — no lock ordering issue,
// this is a leaf read).
func (f *Front) sessionCwdLocked(sessionID string) string {
	f.cwdMu.Lock()
	defer f.cwdMu.Unlock()

	return f.cwds[sessionID]
}
