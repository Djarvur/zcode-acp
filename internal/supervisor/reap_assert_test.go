package supervisor_test

import (
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

const (
	// reapedDeadline bounds the monitor-reap wait shared by the exec-chain
	// and process-lifecycle tests.
	reapedDeadline = 3 * time.Second
	// spawnedChildDeadline bounds the wait for the SpawnFunc's delivery.
	spawnedChildDeadline = 5 * time.Second
)

// waitSpawnedChild waits for the supervision spawn seam to deliver the child.
func waitSpawnedChild(t *testing.T, spawnCh <-chan *supervisor.Child) *supervisor.Child {
	t.Helper()

	select {
	case child := <-spawnCh:
		return child
	case <-time.After(spawnedChildDeadline):
		t.Fatalf("no child delivered by the spawn path within %v", spawnedChildDeadline)

		return nil
	}
}

// assertReaped asserts the monitor reaped the child (Done closed, Exited
// delivered) — the "Wait reaps the child" half of the zero-survivors proof.
func assertReaped(t *testing.T, child *supervisor.Child) {
	t.Helper()

	select {
	case <-child.Done:
	case <-time.After(reapedDeadline):
		t.Fatalf("child was not reaped within %v after the staged shutdown", reapedDeadline)
	}

	select {
	case <-child.Exited:
	case <-time.After(reapedDeadline):
		t.Fatal("the exit monitor never delivered the exit info")
	}
}
