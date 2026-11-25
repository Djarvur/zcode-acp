//go:build linux

package supervisor

import "syscall"

// statfsType normalizes Linux's int64 Statfs_t.Type at the MountOpts seam.
func statfsType(st *syscall.Statfs_t) int64 {
	return st.Type
}
