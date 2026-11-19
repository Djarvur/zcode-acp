package zcode

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

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

// pendingCall is one in-flight request awaiting its reply envelope. The
// reply channel is capacity 1: resolve never blocks the read loop.
type pendingCall struct {
	reply chan Envelope
}

func newPendingCall() *pendingCall {
	return &pendingCall{reply: make(chan Envelope, 1)}
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
func (p *pendingTable) register(id *ID) <-chan Envelope {
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
	call.reply <- env
}

// Call sends method with params on the ordered queue lane and waits for the
// matching reply, correlated by type-tagged canonical id (dialect §1: ids
// are string-or-int and replies echo the request id verbatim). A wire error
// reply surfaces as a *CallError; ctx cancellation deregisters the pending
// entry. Outbound ids are integers, counted per Conn like the probe's.
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
		if res.Error != nil {
			return nil, &CallError{Err: res.Error}
		}

		return res.Result, nil
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
