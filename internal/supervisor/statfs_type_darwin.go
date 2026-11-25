//go:build darwin

package supervisor

import "syscall"

// statfsType normalizes Darwin's uint32 Statfs_t.Type to the int64 seam.
func statfsType(st *syscall.Statfs_t) int64 {
	return int64(st.Type)
}
