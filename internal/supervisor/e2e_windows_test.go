//go:build windows

package supervisor_test

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// reapedPoll paces the zero-survivor poll.
const reapedPoll = 10 * time.Millisecond

// e2eAssertTreeGone proves the fake app-server process is gone after the Job
// Object teardown. KILL_ON_JOB_CLOSE covers descendants; the Windows
// process-tree test pins that half of the contract separately.
func e2eAssertTreeGone(t *testing.T, child *supervisor.Child) {
	t.Helper()

	deadline := time.Now().Add(reapedDeadline)
	pid := uint32(child.PgID)

	for {
		handle, err := windows.OpenProcess(
			windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid,
		)
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return
		}
		if err != nil {
			t.Fatalf("query fake app-server pid %d: %v", child.PgID, err)
		}
		_ = windows.CloseHandle(handle)

		if time.Now().After(deadline) {
			t.Fatalf("the fake app-server (pid %d) survived the staged shutdown", child.PgID)
		}

		time.Sleep(reapedPoll)
	}
}
