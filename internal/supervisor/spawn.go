// Spawn contract (BRIDGE-03 part 2, plan 03-07): the canonical argv line and
// child environment from docs/protocol/install-discovery.md :48-52, plus
// SpawnChild — the process-group start (Setpgid) with all three stdio pipes
// wired and a capped stderr drain. RED stage: this file is a skeleton whose
// only job is to let the contract tests fail on their planned assertions;
// the GREEN step of Task 1 replaces every stub below.

package supervisor

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

// Child is one spawned app-server child process and the handles its
// supervisor needs: the protocol pipes, the process-group id, and the exit
// monitor's delivery. GREEN fills the spawn; the shape is the Task-1
// contract.
type Child struct {
	// Cmd is the exec handle (Process, Wait bookkeeping owned by the
	// monitor inside SpawnChild — never Wait twice).
	Cmd *exec.Cmd
	// Stdin is the protocol's write side; the supervisor holds it open and
	// closing it is the first stage of the shutdown sequence (the child
	// drains ~100 ms on EOF by design, dialect §1 :30).
	Stdin io.WriteCloser
	// Stdout is the protocol's read side — handed to the codec Conn.
	Stdout io.ReadCloser
	// PgID is the child's process-group id (Setpgid: the child is its own
	// group leader, so -PgID addresses the whole tree incl. grandchildren).
	PgID int
	// Exited delivers exactly one ExitInfo when the monitor's Wait has
	// reaped the child (buffered, so the monitor never blocks).
	Exited <-chan ExitInfo
	// StderrDone closes when the capped stderr pump has finished — the
	// Wait-ordering guard of PITFALL 3 (never reap before pumps finish).
	StderrDone <-chan struct{}
}

// ExitInfo is the monitor's one delivery: the reaped child's Wait error
// (nil on a clean exit; *exec.ExitError carries the exit status).
type ExitInfo struct {
	Err error
}

// SpawnOpts tunes one SpawnChild call. Zero values are production defaults.
type SpawnOpts struct {
	// WaitDelay bounds exec.Cmd.Wait's wait for pipe readers after the
	// process exits (stdlib semantics; 0 takes the production default).
	WaitDelay time.Duration
	// StderrCap caps the child stderr bytes forwarded to the wrapper's
	// stderr (0 takes the 64 KiB probe cap of PITFALL 2).
	StderrCap int
}

// errSpawnSkeleton marks the RED-stage skeleton. GREEN replaces it with the
// real spawn contract.
var errSpawnSkeleton = errors.New("zcode: supervisor spawn skeleton: not implemented")

// SpawnArgv returns the canonical spawn argv for an Install
// (install-discovery.md :48-52). RED skeleton: input deliberately unobserved.
func SpawnArgv(_ Install) []string {
	return nil
}

// ChildEnv builds the child environment from parentEnv via the explicit
// allowlist (D-03 / PITFALL 9 / T-3-08). RED skeleton: inputs deliberately
// unobserved.
func ChildEnv(_ Install, _ []string) []string {
	return nil
}

// SpawnChild spawns the child described by argv under env and owns its
// lifecycle plumbing. RED skeleton: the inputs are deliberately unobserved.
func SpawnChild(_ context.Context, _, _ []string, _ SpawnOpts) (*Child, error) {
	return nil, errSpawnSkeleton
}
