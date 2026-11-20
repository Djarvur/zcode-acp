package core_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/core"
	"github.com/Djarvur/zcode-acp/internal/turn"
)

// SessionRegistry unit tests (plan 04-02, RESEARCH Open Question 5
// resolution): register/get/remove round-trips, the loud lookup-miss
// sentinel, last-wins re-registration, and a -race concurrency smoke.

// registryGoroutines and registryOpsPerG bound the concurrency smoke.
const (
	registryGoroutines = 8
	registryOpsPerG    = 25
)

// fixedSessionID is the synthetic session id the lifecycle test rides.
const fixedSessionID = "sess_1"

// newMachine builds a detached turn machine for identity assertions —
// the registry stores *turn.Machine pointers, and NewMachine tolerates a
// nil conn (no traffic ever flows in these tests).
func newMachine(sessionID string) *turn.Machine {
	return turn.NewMachine(nil, sessionID, nil)
}

// TestRegistryRegisterGetRemove drives the lifecycle: a registered
// session resolves to the SAME machinery with its flags, re-registration
// is last-wins, a lookup miss fails loud with ErrNoSession, and Remove
// makes the id miss again (Remove of an unknown id is a no-op).
func TestRegistryRegisterGetRemove(t *testing.T) {
	r := core.NewSessionRegistry()

	m1 := newMachine(fixedSessionID)
	r.Register(core.Session{ID: fixedSessionID, Machine: m1, Subscribed: true})

	got, err := r.Get(fixedSessionID)
	if err != nil {
		t.Fatalf("Get(sess_1): %v", err)
	}

	if got.ID != fixedSessionID || got.Machine != m1 || !got.Subscribed {
		t.Fatalf("Get(%s) = %+v, want the registered session (same machine pointer, subscribed)", fixedSessionID, got)
	}

	m1b := newMachine(fixedSessionID)
	r.Register(core.Session{ID: fixedSessionID, Machine: m1b})

	got, err = r.Get(fixedSessionID)
	if err != nil {
		t.Fatalf("Get(sess_1) after re-register: %v", err)
	}

	if got.Machine != m1b {
		t.Fatal("re-registration must be last-wins: Get returned the stale machine")
	}

	if _, err := r.Get("sess_missing"); !errors.Is(err, core.ErrNoSession) {
		t.Fatalf("Get(sess_missing) = %v, want the loud ErrNoSession sentinel", err)
	}

	r.Remove(fixedSessionID)

	if _, err := r.Get(fixedSessionID); !errors.Is(err, core.ErrNoSession) {
		t.Fatalf("Get(sess_1) after Remove = %v, want ErrNoSession", err)
	}

	r.Remove("sess_missing") // must be a no-op, never a panic
}

// TestRegistryConcurrentAccess hammers register/get/remove from parallel
// goroutines under -race; after the join, every id is removed and misses.
func TestRegistryConcurrentAccess(t *testing.T) {
	r := core.NewSessionRegistry()

	var wg sync.WaitGroup

	for g := range registryGoroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			sessionID := fmt.Sprintf("sess_%d", g)

			for range registryOpsPerG {
				r.Register(core.Session{ID: sessionID, Machine: newMachine(sessionID)})

				if _, err := r.Get(sessionID); err != nil {
					t.Errorf("Get(%s): %v", sessionID, err)
				}

				r.Remove(sessionID)
			}
		}()
	}

	wg.Wait()

	if _, err := r.Get("sess_0"); !errors.Is(err, core.ErrNoSession) {
		t.Fatalf("Get(sess_0) after the join = %v, want ErrNoSession (every goroutine removed its id)", err)
	}
}
