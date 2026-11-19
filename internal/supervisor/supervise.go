// Supervisor (BRIDGE-03 part 2, plan 03-07): health watch, staged shutdown,
// group kill, and the crash-window restart policy over one SpawnChild'd
// app-server child. RED stage: this file is a skeleton whose only job is to
// let the lifecycle tests fail on their planned assertions; the GREEN step
// of Task 2 replaces every stub below.

package supervisor

import (
	"context"
	"errors"
	"time"
)

// Policy is the crash-window restart configuration (D-03: 3 attempts within
// 60 s, then fail loud; RESEARCH Pitfall 7: the sliding window resets on a
// stable run). Zero fields take the production defaults.
type Policy struct {
	// CrashLimit is the max consecutive in-window crashes before fail loud.
	CrashLimit int
	// CrashWindow is the sliding window; a stable run longer than the window
	// resets the crash counter.
	CrashWindow time.Duration
	// BaseBackoff seeds the exponential restart backoff (BaseBackoff <<
	// (crashes-1), ARCHITECTURE Pattern 4).
	BaseBackoff time.Duration
	// Grace is the shutdown grace between stdin end and the group SIGKILL.
	Grace time.Duration
}

// DefaultPolicy is the production policy (D-03).
func DefaultPolicy() Policy {
	return Policy{}
}

// SpawnFunc spawns one fresh child; the restart loop calls it again per
// attempt.
type SpawnFunc func(ctx context.Context) (*Child, error)

// Supervisor owns one SpawnChild'd app-server child across restarts.
type Supervisor struct{}

// NewSupervisor builds a Supervisor over the spawn seam and policy.
func NewSupervisor(spawn SpawnFunc, policy Policy) *Supervisor {
	_, _ = spawn, policy

	return &Supervisor{}
}

// Start supervises the child until ctx is done. RED skeleton.
func (s *Supervisor) Start(ctx context.Context) error {
	_ = ctx

	return errSuperviseSkeleton
}

// errSuperviseSkeleton marks the RED-stage skeleton.
var errSuperviseSkeleton = errors.New("zcode: supervisor supervise skeleton: not implemented")
