package interact

import (
	"sync"
	"time"
)

// dedupeKey is the re-announce correlation key (05-RESEARCH.md Pattern 2):
// every re-announce carries a NEW wire id, so the stable key is the
// (method, params.requestId) pair — never the wire envelope id (Pitfall 1:
// a per-wire-id dedupe map grows one entry per second per pending ask).
type dedupeKey struct {
	method    string
	requestID string
}

// dedupeTable is the bounded re-announce correlation table — the
// call.go pendingTable discipline (one mutex, delete-under-lock
// exactly-once) keyed on (method, params.requestId) with per-entry TTL
// eviction. RED skeleton: zero-value behavior so the contract test
// compiles and fails on its first unmet assertion.
type dedupeTable struct {
	mu      sync.Mutex
	entries map[dedupeKey]time.Time
}

func newDedupeTable() *dedupeTable {
	return &dedupeTable{entries: make(map[dedupeKey]time.Time)}
}

// claim records the (method, requestId) pair and reports whether the
// caller joined an existing entry (true) or is the fresh primary (false).
func (t *dedupeTable) claim(_, _ string, _ time.Duration) bool {
	return false
}

// release evicts the entry on answer.
func (t *dedupeTable) release(_, _ string) {}

// len reports the pending entry count.
func (t *dedupeTable) len() int {
	return 0
}
