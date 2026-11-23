package acpfront

import (
	"context"
	"errors"
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

	// flushCancel cancels the drain's flush context. The context itself is
	// DETACHED from the turn context on purpose (captured by the emit
	// closure, never a struct field — containedctx): release cancels the
	// turn context to wake the drain goroutine BEFORE the final flush, so a
	// turn-derived child would already be dead (context canceled cascades to
	// children instantly) exactly when the flush needs it — the queued
	// deltas would be silently lost (an SDK sink honors its ctx). The flush
	// context lives from turn start until release's flush completes:
	// releaseFlush fires only AFTER the bounded flush wait — never before —
	// so no sink write survives the teardown and none is refused while the
	// flush is still legal.
	flushCancel context.CancelFunc

	mu      sync.Mutex
	queue   []pumpEntry
	bytes   int
	writing bool  // a note is mid-write to the sink (drain goroutine only)
	breach  error // non-nil → the ceiling was hit; every later enqueue is refused
	notify  chan struct{}
	drained chan struct{} // signaled whenever the queue reaches empty
	done    chan struct{}

	// messages counts the agent_message_chunk notes this turn's pump has
	// ACCEPTED (enqueued) — the QA-03 completion-side reconcile's counter
	// (reconcile.go): a completed turn whose count is still zero carries its
	// reply text only in the backend's message store, so the turn-end path
	// reconciles it; a non-zero count means deltas streamed and the
	// reconcile must stay silent (no duplicate). Counting at the enqueue
	// boundary (not the sink write) is the conservative duplicate guard:
	// every accepted chunk rides the release flush even when the reconcile
	// reads mid-drain. The pump is constructed fresh per acquireTurn, so the
	// counter starts at zero each turn — that construction IS the reset.
	messages int
}

// newTurnPump builds the pump for one turn and starts its drain goroutine
// on the turn context. The emitter closure it installs writes each note to
// sink under the pump's detached flush context (alive through the release
// flush).
func newTurnPump( //nolint:contextcheck // the flush ctx is detached BY CONTRACT (the
	// flushCancel field doc): release cancels the TURN context before the
	// final flush, so deriving the flush context from it would kill the
	// flush's own writes exactly when they matter — the detached parent is
	// the fix, bounded by release's flushCancel.
	turnCtx context.Context,
	sessionID string,
	sink acpport.UpdateSink,
	cancel context.CancelFunc,
	ceiling int,
) *turnPump {
	// Detached flush context (see the flushCancel field doc): the parent
	// cancel at release must not kill the flush's own writes mid-flush. The
	// context is captured by the emit closure below — never stored on the
	// pump (containedctx house rule).
	flushCtx, flushCancel := context.WithCancel(context.Background())
	p := &turnPump{
		cancel:      cancel,
		ceiling:     ceiling,
		flushCancel: flushCancel,
		notify:      make(chan struct{}, 1),
		drained:     make(chan struct{}, 1),
		done:        make(chan struct{}),
	}

	p.emit = func(note acpport.SessionNote) error {
		note.SessionID = sessionID

		return sink(flushCtx, note)
	}

	go p.run(turnCtx)

	return p
}

// releaseFlush cancels the flush context — release calls it after the
// bounded flush wait, so no sink write survives the turn's teardown.
func (p *turnPump) releaseFlush() {
	p.flushCancel()
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
// prompt answer carries the honest outcome. The `writing` flag covers each
// emit so barrier() can't observe an empty queue while a write is still in
// flight; reaching empty signals `drained` once per drain pass.
func (p *turnPump) drain() {
	for {
		p.mu.Lock()

		if len(p.queue) == 0 {
			p.mu.Unlock()

			p.signalDrained()

			return
		}

		entry := p.queue[0]
		p.queue = p.queue[1:]
		p.writing = true
		p.mu.Unlock()

		// The write rides flushCtx — detached from the turn context and
		// alive through the release flush (see the field doc). A failed
		// write (a stalled or closing ACP wire) is recorded structurally —
		// never with params/content — and the drain continues; the prompt
		// answer carries the honest outcome.
		err := p.emit(entry.note)

		p.mu.Lock()
		p.writing = false
		p.mu.Unlock()

		if err != nil {
			slog.Warn("acpfront: session update write failed", "err", err)
		}
	}
}

// signalDrained pokes whoever waits for an empty queue (never blocks —
// the channel is buffered to one and the waiter re-checks).
func (p *turnPump) signalDrained() {
	select {
	case p.drained <- struct{}{}:
	default:
	}
}

// barrier waits bounded until every note enqueued so far has been handed
// to the sink — the wire-order handoff for the permission ask: the ask
// frame must land AFTER the tool frames the backend sent before the
// interaction request (started card before ask, fullcycle lock). Past the
// bound the barrier gives up (a stalled sink must not stall the ask) and
// the ask proceeds with best-effort ordering.
func (p *turnPump) barrier(bound time.Duration) {
	deadline := time.After(bound)

	for {
		p.mu.Lock()
		quiet := len(p.queue) == 0 && !p.writing
		p.mu.Unlock()

		if quiet {
			return
		}

		select {
		case <-p.drained:
		case <-deadline:
			return
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

// errReconcileDuplicate is enqueueIfNoMessages' refusal sentinel: the pump
// already accepted a message chunk, so the stored text would duplicate it
// on the client (delta-review WR-03).
var errReconcileDuplicate = errors.New("acpfront: reconcile emit skipped — a message chunk is already queued")

// enqueueIfNoMessages enqueues the stored-reply note ATOMICALLY with the
// message-count check (delta-review WR-03): a check-then-act across two
// lock acquisitions could admit a late delta BETWEEN the check and the
// enqueue, duplicating the reply. One acquisition decides both — zero
// accepted messages enqueues (the counter bumps in appendLocked); any
// answers the refusal sentinel with nothing queued. The breach/ceiling
// discipline is enqueue's, verbatim.
func (p *turnPump) enqueueIfNoMessages(note acpport.SessionNote) error {
	size := noteBytes(note)

	p.mu.Lock()

	if p.breach != nil {
		err := p.breach
		p.mu.Unlock()

		return err
	}

	if p.messages != 0 {
		p.mu.Unlock()

		return errReconcileDuplicate
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

	p.appendLocked(note, size)
	p.mu.Unlock()
	p.signal()

	return nil
}

// appendLocked queues one note and adds its weight; the caller holds mu.
// Message chunks bump the reconcile counter here — the single acceptance
// choke point every enqueued note passes (coalescing never drops or merges
// message chunks: they are not tool patches).
func (p *turnPump) appendLocked(note acpport.SessionNote, size int) {
	p.queue = append(p.queue, pumpEntry{note: note, size: size})
	p.bytes += size

	if note.Kind == acpport.NoteMessage {
		p.messages++
	}
}

// messageCount snapshots the turn's accepted agent_message_chunk count —
// the completion-side reconcile's counter read (reconcile.go).
func (p *turnPump) messageCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.messages
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
	case acpport.NoteCatalog:
		size := perFrameAllowance
		for _, item := range note.Catalog {
			size += len(item.Name) + len(item.Description) + len(item.Hint)
		}

		return size
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
// — a second prompt while a turn is active, refused by the caller as the
// -32003 session-busy error (acpport.NewSessionBusy). A non-nil error means
// the
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

	// The WR-03 pump: the emitter closure (sink + session id, written under
	// the pump's flushCtx) drains at wire pace on the pump goroutine; the
	// tap's enqueue never blocks the backend read loop.
	pump := newTurnPump(turnCtx, sessionID, sink, turnCancel, ceiling)

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
	f.turnStamp = 0
	f.turnCwd = f.sessionCwdLocked(sessionID)
	f.turnDiffs = make(map[string]acpport.ToolDiff)

	release := func() {
		f.turnMu.Lock()

		f.turnActive = false
		f.turnSession = ""
		f.turnDone = nil
		f.turnEmit = nil
		f.turnID = ""
		f.turnStamp = 0
		f.turnCwd = ""
		f.turnDiffs = nil
		pump := f.turnPump
		f.turnPump = nil

		f.turnMu.Unlock()

		// Outside the lock. Cancel first: it wakes the pump's select (and
		// the fan-in goroutine's turnCtx.Done arm); the pump's final drain
		// delivers whatever was legally queued — the flush-before-stopReason
		// property lives HERE, inside release, before Prompt's stopReason is
		// returned. The drain's writes ride flushCtx (still alive — that is
		// the point), so the queued deltas reach the client even though the
		// turn context itself is already canceled. The wait is bounded: a
		// sink parked past the bound abandons the drain so Prompt can never
		// hang on the ACP wire.
		turnCancel()
		pump.wait(pumpReleaseWait)

		// Only now kill the flush context — no sink write survives the
		// turn's teardown, and none is refused before the flush finished.
		pump.releaseFlush()

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
