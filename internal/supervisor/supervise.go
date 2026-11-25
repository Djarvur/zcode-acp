// Supervisor (BRIDGE-03 part 2, plan 03-07): health watch, staged shutdown,
// group kill, and the crash-window restart policy over one SpawnChild'd
// app-server child (ARCHITECTURE.md Pattern 4 :238-268 — the supervisor
// anchor — plus the probe's validated shutdown sequence
// experiments/app-server-probe.mjs :638-653 and its unconditional SIGKILL
// backstop :667-674). Health = stdin liveness + child exit: the supervisor
// holds the child's stdin open (closing it is the deliberate shutdown stage 1)
// and watches the exit — there is NO protocol liveness frame (Phase 1 gap
// note, D-03) and the supervisor does not invent one. The crash window is
// D-03's policy (3 attempts within 60 s, then fail loud naming the crash
// count and window) with the RESEARCH-Pitfall-7 sliding-window reset both
// analogs lack: a stable run longer than the window resets the counter.

package supervisor

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Policy defaults (D-03 + probe facts).
const (
	// defaultCrashLimit is D-03's crash-window limit (the codex-acp policy,
	// not its env-var names — RESEARCH State of the Art).
	defaultCrashLimit = 3
	// defaultCrashWindow is the sliding 60 s window of D-03.
	defaultCrashWindow = 60 * time.Second
	// defaultBaseBackoff seeds the exponential restart backoff
	// (BaseBackoff << (crashes-1), ARCHITECTURE Pattern 4).
	defaultBaseBackoff = 1 * time.Second
	// defaultGrace is the shutdown grace between stdin end and the group
	// SIGKILL — the probe's validated GRACE_MS = 4000.
	defaultGrace = 4 * time.Second

	// shutdownReapDeadline bounds the post-kill reap wait (WaitDelay inside
	// SpawnChild caps the Wait itself; this guard only bounds OUR wait on it).
	shutdownReapDeadline = 10 * time.Second
)

// Policy is the crash-window restart configuration (D-03: 3 attempts within
// 60 s, then fail loud; RESEARCH Pitfall 7: the sliding window resets on a
// stable run longer than the window). Zero fields take the production
// defaults at Start time.
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
	return Policy{
		CrashLimit:  defaultCrashLimit,
		CrashWindow: defaultCrashWindow,
		BaseBackoff: defaultBaseBackoff,
		Grace:       defaultGrace,
	}
}

// withDefaults resolves zero fields to the production defaults.
func (p Policy) withDefaults() Policy {
	if p.CrashLimit <= 0 {
		p.CrashLimit = defaultCrashLimit
	}

	if p.CrashWindow <= 0 {
		p.CrashWindow = defaultCrashWindow
	}

	if p.BaseBackoff <= 0 {
		p.BaseBackoff = defaultBaseBackoff
	}

	if p.Grace <= 0 {
		p.Grace = defaultGrace
	}

	return p
}

// SpawnFunc spawns one fresh child; the restart loop calls it again per
// attempt.
type SpawnFunc func(ctx context.Context) (*Child, error)

// Supervisor owns one SpawnChild'd app-server child across restarts: it
// spawns through the seam, watches health, restarts within the crash window,
// and stage-shuts-down the whole process group on exit or ctx done.
type Supervisor struct {
	spawn  SpawnFunc
	policy Policy
	log    *slog.Logger
}

// NewSupervisor builds a Supervisor over the spawn seam and policy.
func NewSupervisor(spawn SpawnFunc, policy Policy) *Supervisor {
	return &Supervisor{spawn: spawn, policy: policy, log: slog.Default()}
}

// Start supervises the child until ctx is done. Intentional shutdown (ctx
// done) returns nil after the staged shutdown; a crash loop past the policy
// limit fails loud naming the crash count and the window; a spawn failure
// fails loud immediately. Health = stdin liveness + child exit — the
// supervisor never invents a protocol liveness frame (D-03).
func (s *Supervisor) Start(ctx context.Context) error {
	policy := s.policy.withDefaults()
	crashes := 0

	for {
		child, spawnErr := s.spawn(ctx)
		if spawnErr != nil {
			return fmt.Errorf("zcode: supervisor: spawn app-server: %w", spawnErr)
		}

		runStart := time.Now()

		select {
		case info := <-child.Exited:
			if time.Since(runStart) > policy.CrashWindow {
				crashes = 0 // sliding-window reset (RESEARCH Pitfall 7)
			}

			crashes++

			if crashes > policy.CrashLimit {
				// Staged shutdown ALWAYS — the fail-loud exit included
				// (review CR-01): returning before the shutdown would strand
				// the crashed child's whole group — Electron grandchildren
				// included (PITFALL 6) — exactly the leak the supervisor
				// exists to contain.
				s.shutdown(child, policy)

				return fmt.Errorf("zcode: supervisor: app-server crash loop: %d crashes within %v: %w",
					crashes, policy.CrashWindow, info.Err)
			}

			s.log.Warn("supervisor: app-server exited; restarting within the crash window",
				"crashes", crashes, "window", policy.CrashWindow, "err", info.Err)

		case <-ctx.Done():
		}

		// Staged shutdown ALWAYS — even a crashed child's group may hold
		// orphaned grandchildren; even a clean-exit child may leave helpers
		// behind (PITFALL 6: never trust stdin-EOF on Electron).
		s.shutdown(child, policy)

		if ctx.Err() != nil {
			return nil //nolint:nilerr // intentional shutdown returns nil per the Start contract
		}

		select {
		case <-time.After(policy.BaseBackoff << (crashes - 1)):
		case <-ctx.Done():
			return nil
		}
	}
}

// shutdown performs the staged teardown (RESEARCH :375-378 order; probe
// :667-674 precedent): close stdin → wait for exit within the grace window →
// UNCONDITIONAL group SIGKILL → bounded reap wait. The kill is unconditional
// because stdin-EOF is never trusted (PITFALL 6): a cleanly-exited child may
// still leave orphaned group members. Idempotent on an already-reaped child
// (Done already closed; ESRCH tolerated).
func (s *Supervisor) shutdown(child *Child, policy Policy) {
	_ = child.Stdin.Close() // stage 1: stdin end (the child drains ~100 ms by design, dialect §1 :30)

	select { // stage 2: wait for exit within the grace window
	case <-child.Done:
	case <-time.After(policy.Grace):
	}

	// Stage 3: the group SIGKILL backstop — best-effort, logged, never
	// propagated (the shutdown contract).
	if err := child.terminateTree(); err != nil {
		s.log.Warn("supervisor: staged shutdown process-tree kill", "pid", child.PgID, "err", err)
	}

	select { // stage 4: the reap completes (never Wait twice — the monitor owns it)
	case <-child.Done:
	case <-time.After(shutdownReapDeadline):
		s.log.Warn("supervisor: staged shutdown reap did not complete", "pgid", child.PgID)
	}
}
