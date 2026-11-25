//go:build !windows

package supervisor_test

import (
	"testing"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// e2eAssertTreeGone proves the fake app-server's process group has no
// survivors (kill(-pgid, 0) → ESRCH).
func e2eAssertTreeGone(t *testing.T, child *supervisor.Child) {
	t.Helper()

	assertGone(t, "the fake app-server process group", -child.PgID)
}
