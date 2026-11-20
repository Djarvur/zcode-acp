package interact

import (
	"sync"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// dedupeGrace extends each pending entry's TTL past its policy deadline:
// the entry releases into a resolution tombstone on answer, and the TTL is
// the backstop for a primary wedged past its own deadline.
const dedupeGrace = 5 * time.Second

// resolutionGrace keeps an answered set's tombstone claimable long enough
// to catch in-flight re-announce frames (a re-announce already written to
// the wire when the first reply resolves lands milliseconds later; the
// server cancels its re-announce timer on resolve, so nothing legitimate
// arrives beyond wire slack). A tombstone join stays silent — the set is
// answered, and any frame to its ids would be dropped server-side anyway.
const resolutionGrace = 2 * time.Second

// dedupeKey is the re-announce correlation key (05-RESEARCH.md Pattern 2):
// every re-announce carries a NEW wire id, so the stable key is the
// (method, params.requestId) pair — never the wire envelope id (Pitfall 1:
// a per-wire-id dedupe map grows one entry per second per pending ask).
type dedupeKey struct {
	method    string
	requestID string
}

// dedupeEntry is one claimed (method, requestId): pending while the
// primary's handler runs, then a resolved tombstone until its TTL elapses
// (in-flight re-announce catch). The value-in-map is the entry's TTL
// expiry — the bounded map's per-entry eviction clock (T-5-01).
type dedupeEntry struct {
	expires  time.Time
	resolved bool
}

// dedupeTable is the bounded re-announce correlation table — the
// call.go pendingTable discipline (one mutex, delete-under-lock
// exactly-once so a claimed key can never double-run) keyed on
// (method, params.requestId). The growth ceiling is one entry per distinct
// requestId — a server that re-announces forever grows the table by one
// entry per distinct requestId, never one per wire frame — and every entry
// is evicted on answer (release), by its per-entry TTL sweep, or with the
// responder process ending (the map dies with it; conn-close eviction
// rides the release that happens when the primary's request context is
// cancelled).
type dedupeTable struct {
	mu      sync.Mutex
	entries map[dedupeKey]dedupeEntry
	now     func() time.Time
}

func newDedupeTable() *dedupeTable {
	return &dedupeTable{
		entries: make(map[dedupeKey]dedupeEntry),
		now:     time.Now,
	}
}

// claim records the (method, requestId) pair and reports whether the
// caller joined an existing entry (true — pending OR resolved tombstone:
// both mean the set is already owned) or is the fresh primary (false).
// Expired entries are swept first — the bounded map's TTL eviction.
func (t *dedupeTable) claim(method, requestID string, ttl time.Duration) bool {
	now := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	t.sweep(now)

	key := dedupeKey{method: method, requestID: requestID}

	if _, joined := t.entries[key]; joined {
		return true
	}

	t.entries[key] = dedupeEntry{expires: now.Add(ttl)}

	return false
}

// release marks the entry resolved on answer — delete-under-lock would
// let an in-flight re-announce re-run the handler and frame a second
// reply, so the entry tombstones instead and its TTL sweep removes it.
// A racing same-key claim after release therefore joins (stays silent),
// never double-answers.
func (t *dedupeTable) release(method, requestID string) {
	now := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	key := dedupeKey{method: method, requestID: requestID}

	if ent, ok := t.entries[key]; ok && !ent.resolved {
		t.entries[key] = dedupeEntry{expires: now.Add(resolutionGrace), resolved: true}
	}
}

// sweep evicts every expired entry. Caller holds mu.
func (t *dedupeTable) sweep(now time.Time) {
	for key, ent := range t.entries {
		if now.After(ent.expires) {
			delete(t.entries, key)
		}
	}
}

// len reports the pending (unresolved) entry count — tombstones are
// resolution bookkeeping, not pending asks.
func (t *dedupeTable) len() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	n := 0

	for _, ent := range t.entries {
		if !ent.resolved {
			n++
		}
	}

	return n
}

// dedupeTTL is a pending entry's time-to-live: the policy deadline plus
// grace. A zero deadline (no deadline policy) uses the router default so
// no entry is immortal.
func dedupeTTL(deadline time.Duration) time.Duration {
	if deadline <= 0 {
		deadline = zcode.RouterDefaultDeadline
	}

	return deadline + dedupeGrace
}
