package zcode_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Single-writer ordering (RESEARCH Pitfall 7): concurrent Call writers
// produce complete, never-interleaved frames on the child side — one line is
// one envelope, always — and every caller receives exactly its own reply, so
// the frame count is exact.
func TestWriterOrdering(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	echoErrs := echoPeer(t, peerEnd)

	ctx, cancel := contextWithTestTimeout(t)
	defer cancel()

	drainCallErrors(t, concurrentCalls(ctx, t, conn, 32))

	select {
	case err := <-echoErrs:
		if err != nil {
			t.Fatalf("recording peer observed corrupt frames (interleaved writes): %v", err)
		}
	default:
	}
}

// Bypass precedence (dialect §1): session/stop issued while the ordered
// queue holds pending writes is on the wire after at most the in-flight
// write — it never queues behind the whole queue. io.Pipe's synchronous
// blocking makes the first queued frame the writer's in-flight write and
// keeps the second parked in the queue until a reader appears; the stop is
// submitted before the reader starts, so the deterministic frame order is
// [in-flight, stop, queued] — the stop never lands third.
func TestStopJumpsQueue(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	ctx, cancel := contextWithTestTimeout(t)
	defer cancel()

	outcomes := issueCalls(ctx, t, conn, 2)

	// Cancellation submitted BEFORE the pipe is read: the bypass lane must
	// already hold the stop frame when the in-flight write completes.
	if err := conn.Stop(testSessionID); err != nil {
		t.Fatalf("stop: %v", err)
	}

	first3 := waitFrames(t, readFrames(peerEnd, 3), 3)

	stopAt := -1

	var calls []zcode.Envelope

	for i, line := range first3 {
		env := decodeWireLine(t, line)

		if env.Method == zcode.MethodSessionStop {
			stopAt = i
			assertStopFrame(t, env)
		} else {
			calls = append(calls, env)
		}
	}

	if stopAt < 0 || stopAt > 1 {
		t.Fatalf("stop frame landed at wire position %d — it queued behind both pending writes: %v",
			stopAt, first3)
	}

	if len(calls) != 2 {
		t.Fatalf("expected two queued call frames among the first three, got %d", len(calls))
	}

	// The queued calls still complete: answer each and collect outcomes.
	answerCalls(t, peerEnd, calls)

	for range 2 {
		oc := waitOutcome(t, outcomes)

		if oc.err != nil {
			t.Fatalf("queued call errored after the bypass: %v", oc.err)
		}
	}
}

// contextWithTestTimeout returns a ctx bounded by the shared test timeout.
func contextWithTestTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	return context.WithTimeout(context.Background(), testTimeout)
}

// waitFrames waits for the slow peer's frame batch, failing on timeout.
func waitFrames(t *testing.T, ch <-chan []string, n int) []string {
	t.Helper()

	select {
	case got := <-ch:
		if len(got) != n {
			t.Fatalf("slow peer read %d frames, want %d", len(got), n)
		}

		return got
	case <-time.After(testTimeout):
		t.Fatalf("slow peer never read %d frames", n)

		return nil
	}
}

// assertStopFrame checks the stop frame's wire shape: request-shaped (id
// present) with the session id in params — the D-02 cancel primitive. The
// params decode as a map: the wire key is the dialect's camelCase
// sessionId, and a struct tag would mis-state it.
func assertStopFrame(t *testing.T, env zcode.Envelope) {
	t.Helper()

	if env.ID == nil {
		t.Fatal("stop frame is not request-shaped (no id)")
	}

	var params map[string]any

	if err := json.Unmarshal(env.Params, &params); err != nil {
		t.Fatalf("decode stop params: %v", err)
	}

	if sid, _ := params["sessionId"].(string); sid != testSessionID {
		t.Fatalf("stop params sessionId = %q, want %q", sid, testSessionID)
	}
}

// answerCalls replies to each call frame on the peer side so the queued
// calls settle (verbatim id echo, dialect §1).
func answerCalls(t *testing.T, peer *pipeEnd, calls []zcode.Envelope) {
	t.Helper()

	for _, env := range calls {
		reply, err := zcode.Encode(&zcode.Envelope{ID: env.ID, Result: json.RawMessage(`{"ok":true}`)})
		if err != nil {
			t.Fatalf("encode answer: %v", err)
		}

		if _, err := peer.Write(reply); err != nil {
			t.Fatalf("write answer: %v", err)
		}
	}
}
