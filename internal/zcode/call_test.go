package zcode_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// echoPeer reads every request the conn writes and answers each with a reply
// that echoes the request id verbatim (dialect §1) and carries the request's
// params back as the result — per-call correlation made visible. Corrupt
// frames (interleaved or truncated writes) surface as errors on the
// returned channel; the channel closes when the stream ends.
func echoPeer(t *testing.T, peer *pipeEnd) <-chan error {
	t.Helper()

	errs := make(chan error, 64)

	go func() {
		defer close(errs)

		sc := bufio.NewScanner(peer.r)
		for sc.Scan() {
			env, err := zcode.Decode(sc.Bytes())
			if err != nil {
				errs <- fmt.Errorf("corrupt frame on the wire: %w (%s)", err, sc.Text())

				continue
			}

			if env.ID == nil || env.Method == "" {
				continue
			}

			reply, err := zcode.Encode(&zcode.Envelope{ID: env.ID, Result: env.Params})
			if err != nil {
				errs <- fmt.Errorf("encode echo reply: %w", err)

				continue
			}

			if _, err := peer.Write(reply); err != nil {
				errs <- fmt.Errorf("write echo reply: %w", err)

				return
			}
		}
	}()

	return errs
}

// Correlation contract: replies correlate by exact id value AND type — each
// concurrent call receives exactly its own reply (echoed params carry the
// caller's marker), never another call's, under -race.
func TestCallReplyEchoesIDVerbatim(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	echoErrs := echoPeer(t, peerEnd)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	drainCallErrors(t, concurrentCalls(ctx, t, conn, 5))

	select {
	case err := <-echoErrs:
		if err != nil {
			t.Fatalf("echo peer observed wire corruption: %v", err)
		}
	default:
	}
}

// Cancellation cleanup: a call whose ctx dies before any reply returns the
// ctx error, the pending entry is deregistered (the late reply is dropped
// silently — no panic, no cross-resolution), and a fresh call correlates
// normally afterwards.
func TestCallCtxCancellationCleanup(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	peerLines := drainPeer(t, peerEnd)

	ctx, cancel := context.WithCancel(context.Background())
	outcomes := issueCalls(ctx, t, conn, 1)
	waitLines(t, peerLines, 1, testTimeout) // registered + on the wire

	cancel()

	select {
	case oc := <-outcomes:
		if !errors.Is(oc.err, context.Canceled) {
			t.Fatalf("cancelled call err = %v, want context.Canceled", oc.err)
		}
	case <-time.After(testTimeout):
		t.Fatal("cancelled call never settled")
	}

	// The late reply for the deregistered id must be dropped silently.
	injectLine(t, peerEnd, `{"id":1,"result":{"late":true}}`)

	// The table must be clean: a fresh call correlates normally (its id is
	// 2 — the counter keeps counting).
	freshCtx, freshCancel := context.WithTimeout(context.Background(), testTimeout)
	defer freshCancel()

	fresh := issueCalls(freshCtx, t, conn, 1)

	lines := waitLines(t, peerLines, 1, testTimeout)
	env := decodeWireLine(t, lines[len(lines)-1])

	if env.ID == nil || !env.ID.IsInt || env.ID.Int != 2 {
		t.Fatalf("fresh call id = %+v, want integer 2", env.ID)
	}

	injectLine(t, peerEnd, fmt.Sprintf(`{"id":%d,"result":{"fresh":true}}`, env.ID.Int))

	select {
	case oc := <-fresh:
		if oc.err != nil {
			t.Fatalf("fresh call errored after a cancellation cleanup: %v", oc.err)
		}

		if string(oc.res) != `{"fresh":true}` {
			t.Fatalf("fresh call result = %s, want {\"fresh\":true}", oc.res)
		}
	case <-time.After(testTimeout):
		t.Fatal("fresh call never settled after a cancellation cleanup")
	}
}

// Stream-death contract (review WR-01): when the child dies mid-call — the
// stream EOFs — in-flight calls resolve with an error PROMPTLY. The caller
// ctx here lives far longer than the resolution bound: only a prompt
// stream-death resolution can pass, so the unfixed conn (nothing fails
// pending calls on read-loop exit) wedges until the bound fires while the
// ctx is still fresh — and the pending entry leaks until the ctx deadline.
func TestCallResolvesOnStreamDeath(t *testing.T) {
	connEnd, peerEnd := connPipes()

	conn := zcode.NewConn(connEnd)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	outcomes := issueCalls(ctx, t, conn, 1)
	peerLines := drainPeer(t, peerEnd)

	waitLines(t, peerLines, 1, testTimeout) // registered + on the wire

	_ = peerEnd.Close() // the child dies mid-call: its pipe ends close

	select {
	case oc := <-outcomes:
		if oc.err == nil {
			t.Fatal("a call against a dead stream must fail, not return a result")
		}

		if errors.Is(oc.err, context.DeadlineExceeded) {
			t.Fatalf("stream-death error %v must not be the caller ctx expiry", oc.err)
		}
	case <-time.After(deadStreamBound):
		t.Fatalf("in-flight call still wedged %v after stream death — it hangs until the caller ctx (%v) expires",
			deadStreamBound, testTimeout)
	}
}

// Close contract (review WR-01): a call pending across Conn.Close resolves
// with an error promptly — Close cancels only the conn's base context,
// which pending calls (per-call caller ctxs) never observe, so only an
// explicit drain of the pending table can settle them.
func TestCallResolvesOnClose(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	outcomes := issueCalls(ctx, t, conn, 1)
	peerLines := drainPeer(t, peerEnd)

	waitLines(t, peerLines, 1, testTimeout) // registered + on the wire

	closeDone := make(chan error, 1)

	go func() {
		closeDone <- conn.Close()
	}()

	select {
	case oc := <-outcomes:
		if oc.err == nil {
			t.Fatal("a call pending across Close must fail, not return a result")
		}

		if errors.Is(oc.err, context.DeadlineExceeded) {
			t.Fatalf("Close error %v must not be the caller ctx expiry", oc.err)
		}
	case <-time.After(deadStreamBound):
		t.Fatalf("in-flight call still wedged %v after Close — Close must fail pending calls", deadStreamBound)
	}

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close reported %v", err)
		}
	case <-time.After(deadStreamBound):
		t.Fatal("Close never returned")
	}
}
