package interact

import (
	"testing"
	"time"
)

// The re-announce dedupe's TTL contract (quick task 261124-tj2): the
// app-server re-announces a pending ask with the same params.requestId until
// somebody answers it, and a human dialog has no policy deadline. The entry
// must therefore stay alive as long as the re-announces keep arriving — the
// TTL is a backstop for a wedged primary, not a cap on the human's thinking
// time. These tests drive claim/release against an injected clock; they never
// sleep.

func TestDedupeJoinRefreshesPendingTTL(t *testing.T) {
	t.Parallel()

	// The fallback TTL for the requestUserInput row (Deadline 0): the field
	// defect's 20 s dialog cadence is exactly this value.
	if got := dedupeTTL(0); got != 20*time.Second {
		t.Fatalf("dedupeTTL(0) = %v, want the 15 s router default + 5 s grace", got)
	}

	now := time.Date(2026, 11, 24, 4, 27, 30, 0, time.UTC)
	tbl := newDedupeTable()
	tbl.now = func() time.Time { return now }

	// Each join must re-arm the 20 s window at its own arrival time: the
	// 38 s claim is the regression — on the unfixed code the entry expired
	// at 20 s and the re-announce became a second primary.
	claims := []struct {
		advance    time.Duration
		wantJoined bool
	}{
		{advance: 0, wantJoined: false},                // first envelope claims the set
		{advance: 19 * time.Second, wantJoined: true},  // inside the first window
		{advance: 19 * time.Second, wantJoined: true},  // past the original expiry, inside the refreshed one
		{advance: 19 * time.Second, wantJoined: true},  // keeps refreshing while re-announces arrive
		{advance: 19 * time.Second, wantJoined: true},  // ...
		{advance: 21 * time.Second, wantJoined: false}, // re-announces stopped: the backstop fires
	}

	for i, c := range claims {
		now = now.Add(c.advance)

		joined := tbl.claim(MethodRequestUserInput, "ask-1", dedupeTTL(0))
		if joined != c.wantJoined {
			t.Fatalf("claim %d at %s: joined = %v, want %v",
				i, now.Format(time.TimeOnly), joined, c.wantJoined)
		}

		if got := tbl.len(); got != 1 {
			t.Fatalf("claim %d at %s: pending = %d, want 1", i, now.Format(time.TimeOnly), got)
		}
	}
}

func TestDedupeTombstoneIsNotRefreshed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 24, 4, 27, 30, 0, time.UTC)
	tbl := newDedupeTable()
	tbl.now = func() time.Time { return now }

	const ttl = 20 * time.Second

	if joined := tbl.claim(MethodRequestUserInput, "ask-released", ttl); joined {
		t.Fatalf("first claim joined an empty table")
	}

	// Answer at +1 s: the entry becomes a resolved tombstone expiring at +3 s.
	now = now.Add(time.Second)
	tbl.release(MethodRequestUserInput, "ask-released")

	if got := tbl.len(); got != 0 {
		t.Fatalf("pending after release = %d, want 0 (a tombstone is not pending)", got)
	}

	// A late re-announce inside the resolution grace joins silently.
	now = now.Add(time.Second)
	if joined := tbl.claim(MethodRequestUserInput, "ask-released", ttl); !joined {
		t.Fatalf("claim inside the resolution grace = fresh primary, want a silent join")
	}

	// Past the grace the set is gone — and a join must not have refreshed
	// the tombstone's expiry (that would delay the sweep indefinitely).
	now = now.Add(2 * time.Second)
	if joined := tbl.claim(MethodRequestUserInput, "ask-released", ttl); joined {
		t.Fatalf("claim after the resolution grace joined, want a fresh primary — the tombstone was refreshed")
	}
}
