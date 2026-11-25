//go:build darwin

package supervisor

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrMountLost keeps the shared run-loop contract available on Windows.
	ErrMountLost = errors.New("zcode: appimage mount lost")
	// ErrAppImageUnsupported reports that packed AppImage backends are a Linux
	// feature. macOS uses an installed ZCode app or an explicit binary path.
	ErrAppImageUnsupported = errors.New(
		"zcode: packed AppImage backends are unsupported on Windows; " +
			"use an installed ZCode desktop or --app-server ZCode.exe",
	)
)

// MountOpts mirrors the Unix mount configuration so callers compile on every
// platform. macOS has no FUSE mount lifecycle and ignores these fields.
type MountOpts struct {
	ReadyTimeout time.Duration
	StopGrace    time.Duration
	ReapDeadline time.Duration
	PollInterval time.Duration
	LossPoll     time.Duration
	Statfs       func(path string) (fsType int64, err error)
}

// DefaultMountOpts returns the zero mount options; macOS never mounts.
func DefaultMountOpts() MountOpts {
	return MountOpts{}
}

// MountFailure reports whether err is the AppImage-unsupported sentinel.
func MountFailure(err error) bool {
	return errors.Is(err, ErrAppImageUnsupported)
}

// Mount is the macOS no-op shape of the shared mount owner. A macOS
// install never produces one; its methods remain nil-safe for the run loop.
type Mount struct{}

// PrepareMount refuses every bundle mount: macOS has no FUSE lifecycle.
func PrepareMount(context.Context, Install, MountOpts) (*Mount, error) {
	return nil, ErrAppImageUnsupported
}

// Install returns the zero Install value; a Windows mount never materializes
// one.
func (m *Mount) Install() Install { return Install{} }

// AppImage returns the empty bundle path; macOS installs are app bundles.
func (m *Mount) AppImage() string { return "" }

// Root returns the empty mount root; macOS installs are app bundles.
func (m *Mount) Root() string { return "" }

// Lost stays nil: macOS has no mount helper whose death could be observed.
func (m *Mount) Lost() <-chan struct{} { return nil }

// Accessible is always false: macOS installs never mount an AppImage.
func (m *Mount) Accessible() bool { return false }

// Guard is nil-safe and reports the unsupported sentinel for a real mount.
func (m *Mount) Guard() error {
	if m == nil {
		return nil
	}

	return ErrAppImageUnsupported
}

// HelperPID returns zero: macOS has no mount helper process.
func (m *Mount) HelperPID() int { return 0 }

// Stop is a no-op: macOS has no mount to stop.
func (m *Mount) Stop() error { return nil }
