//go:build !windows

package supervisor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// unixProcessTree addresses the child's process group. Setpgid makes the
// child's pid the group id, so terminating the group includes grandchildren.
type unixProcessTree struct {
	mu       sync.Mutex
	pid      int
	assigned bool
}

//nolint:ireturn // the platform constructor returns the shared processTree interface
func newProcessTree() (processTree, error) {
	return &unixProcessTree{}, nil
}

func configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func (t *unixProcessTree) assign(proc *os.Process) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pid = proc.Pid
	t.assigned = true

	return nil
}

func (t *unixProcessTree) resume() error {
	return nil
}

func (t *unixProcessTree) terminate() error {
	t.mu.Lock()
	pid := t.pid
	assigned := t.assigned
	t.mu.Unlock()

	if !assigned || pid == 0 {
		return nil
	}

	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("zcode: group kill %d: %w", pid, err)
	}

	return nil
}

func (t *unixProcessTree) close() {}
