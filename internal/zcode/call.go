package zcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// ErrConnClosed reports a call that can never complete: the stream died
// (the read loop exited) or the conn closed before the reply arrived.
// errors.Is matches it through the returned call error — match the
// sentinel, never the message text (the dialect §6 matching rule applied
// to Go errors).
var ErrConnClosed = errors.New("zcode: connection closed")

// CallError is a request that completed with a wire error reply. Consumers
// match the numeric Code, never the message text (dialect §6 matching rule:
// messages are not a stable contract).
type CallError struct {
	Err *WireError
}

// Error implements the error interface.
func (e *CallError) Error() string {
	return fmt.Sprintf("zcode: wire error %d: %s", e.Err.Code, e.Err.Message)
}

// callResult is one pending call's terminal delivery: the wire reply
// envelope (resolve), or a drain error — the stream died or the conn
// closed, so no reply can ever arrive (WR-01).
type callResult struct {
	env Envelope
	err error
}

// pendingCall is one in-flight request awaiting its reply envelope. The
// reply channel is capacity 1: resolve never blocks the read loop.
type pendingCall struct {
	reply chan callResult
}

func newPendingCall() *pendingCall {
	return &pendingCall{reply: make(chan callResult, 1)}
}

// pendingTable correlates replies by type-tagged canonical id (dialect §1:
// replies echo the request id verbatim, and "5" the string vs 5 the int are
// different wire values). Method-bearing frames NEVER enter this table —
// the WR-02 demux rule (RESEARCH Pitfall 8) lives in the read loop, not
// here.
type pendingTable struct {
	mu    sync.Mutex
	calls map[string]*pendingCall
}

func newPendingTable() *pendingTable {
	return &pendingTable{calls: make(map[string]*pendingCall)}
}

// register creates the reply channel for a new pending id.
func (p *pendingTable) register(id *ID) <-chan callResult {
	call := newPendingCall()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls[id.Key()] = call

	return call.reply
}

// remove deregisters a pending id (ctx-cancellation cleanup).
func (p *pendingTable) remove(id *ID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.calls, id.Key())
}

// resolve delivers a reply envelope to the pending call registered under the
// envelope's canonical id key and deregisters it — one reply resolves one
// call, and the reply channel's capacity-1 buffer keeps the read loop
// non-blocking. An unknown id (a late reply after ctx cancellation, or an
// unsolicited one) is dropped: the demux only routes result/error frames
// here — method-bearing frames never enter the table (WR-02, RESEARCH
// Pitfall 8).
func (p *pendingTable) resolve(env Envelope) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := env.ID.Key()

	call, ok := p.calls[key]
	if !ok {
		return
	}

	delete(p.calls, key)
	call.reply <- callResult{env: env}
}

// drainAll fails and removes EVERY pending call — the stream died or the
// conn closed, so no reply will ever arrive (WR-01). Each entry resolves
// exactly once: the delete runs under the same lock resolve holds, so a
// call drained here can never also consume a late wire reply, and a wire
// reply that resolved first simply leaves nothing to drain.
func (p *pendingTable) drainAll(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, call := range p.calls {
		delete(p.calls, key)
		call.reply <- callResult{err: err}
	}
}

// Call sends method with params on the ordered queue lane and waits for the
// matching reply, correlated by type-tagged canonical id (dialect §1: ids
// are string-or-int and replies echo the request id verbatim). A wire error
// reply surfaces as a *CallError; ctx cancellation deregisters the pending
// entry; stream death (read-loop exit) and Close drain the pending table,
// failing every in-flight call with an error matching ErrConnClosed
// (WR-01) — a dead stream never wedges a call until its own ctx expires.
// Outbound ids are integers, counted per Conn like the probe's.
func (c *Conn) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := &ID{Int: c.nextID.Add(1), IsInt: true}

	env := &Envelope{ID: id, Method: method}

	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("zcode: call %s: marshal params: %w", method, err)
		}

		env.Params = p
	}

	reply := c.pending.register(id)

	if err := c.out.enqueue(env); err != nil {
		c.pending.remove(id)

		return nil, fmt.Errorf("zcode: call %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		c.pending.remove(id)

		return nil, fmt.Errorf("zcode: call %s: %w", method, ctx.Err())
	case res := <-reply:
		if res.err != nil {
			return nil, fmt.Errorf("zcode: call %s: %w", method, res.err)
		}

		if res.env.Error != nil {
			return nil, &CallError{Err: res.env.Error}
		}

		return res.env.Result, nil
	}
}

// Notify sends method as a notification (no id, no reply expected — dialect
// §1: notifications need no reply) on the ordered queue lane.
func (c *Conn) Notify(method string, params any) error {
	env := &Envelope{Method: method}

	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("zcode: notify %s: marshal params: %w", method, err)
		}

		env.Params = p
	}

	if err := c.out.enqueue(env); err != nil {
		return fmt.Errorf("zcode: notify %s: %w", method, err)
	}

	return nil
}

// Stop sends session/stop for sessionID on the bypass lane (dialect §1: the
// bypass set is exactly session/stop, workspace/cancelGenerateText, and the
// MCP-UI concurrent set — cancellation must never queue behind ordinary
// traffic; D-02 makes this the turn-cancel primitive). The frame is
// request-shaped with a conn-assigned id; the ack reply is deliberately
// unobserved — cancellation is fire-and-forget (the turn machine watches
// completion signals, not the stop ack), so the reply resolves nothing and
// is dropped.
func (c *Conn) Stop(sessionID string) error {
	params, err := json.Marshal(map[string]any{"sessionId": sessionID})
	if err != nil {
		return fmt.Errorf("zcode: stop %s: marshal params: %w", sessionID, err)
	}

	env := &Envelope{
		ID:     &ID{Int: c.nextID.Add(1), IsInt: true},
		Method: MethodSessionStop,
		Params: params,
	}

	if err := c.out.enqueueBypass(env); err != nil {
		return fmt.Errorf("zcode: stop %s: %w", sessionID, err)
	}

	return nil
}

// Reply answers a server→host request id with result on the bypass lane
// (dialect §1: reverse-request responses bypass the queue — answering them
// from the queue would deadlock the pairing). The id echoes the request's
// id verbatim.
func (c *Conn) Reply(id *ID, result any) error {
	env := &Envelope{ID: id}

	if result != nil {
		r, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("zcode: reply %s: marshal result: %w", id.Key(), err)
		}

		env.Result = r
	}

	if err := c.out.enqueueBypass(env); err != nil {
		return fmt.Errorf("zcode: reply %s: %w", id.Key(), err)
	}

	return nil
}

// ReplyError answers a server→host request id with a wire error on the
// bypass lane (the probe precedent: answering unknown host-method requests
// with -32601 did not break create flows — dialect §3).
func (c *Conn) ReplyError(id *ID, werr *WireError) error {
	env := &Envelope{ID: id, Error: werr}

	if err := c.out.enqueueBypass(env); err != nil {
		return fmt.Errorf("zcode: reply error %s: %w", id.Key(), err)
	}

	return nil
}
