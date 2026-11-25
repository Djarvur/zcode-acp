//go:build darwin

package interact

import (
	"context"
	"errors"
)

// errKeyringDarwinUnsupported is the honest macOS stub: the Keychain-backed
// store is the Darwin port's work; until then the credential chain declines
// instead of shelling out to a Linux tool.
var errKeyringDarwinUnsupported = errors.New( //nolint:gochecknoglobals // sentinel
	"keyring probe: macOS Keychain support is not implemented yet")

// NewPlatformKeyLoader returns the Darwin loader stub. The macOS port will
// replace it with a Security-framework Keychain read.
//
//nolint:ireturn // platform factory returns the shared KeyLoader interface
func NewPlatformKeyLoader(string, []string) KeyLoader {
	return darwinKeyLoader{}
}

type darwinKeyLoader struct{}

// LoadKey declines: the Keychain store is not wired yet.
func (darwinKeyLoader) LoadKey(context.Context) ([]byte, error) {
	return nil, errKeyringDarwinUnsupported
}

// NewPlatformKeyringProber returns the Darwin probe stub: the honest
// not-implemented arm, never a secret-tool shell-out.
func NewPlatformKeyringProber(string, []string) KeyringProber {
	return func(context.Context) error { return errKeyringDarwinUnsupported }
}
