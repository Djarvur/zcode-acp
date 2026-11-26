//go:build darwin

package interact

// NewPlatformKeyLoader returns the Darwin credential loader: the Keychain
// generic-password read over the desktop's credentials path.
//
//nolint:ireturn // platform factory returns the shared KeyLoader interface
func NewPlatformKeyLoader(credentialsPath string, env []string) KeyLoader {
	return NewSecurityKeyLoader(DefaultSecurityBin, credentialsPath, env)
}

// NewPlatformKeyringProber returns the Darwin proactive probe: the Keychain
// lookup contract with neither missing nor locked items hidden.
func NewPlatformKeyringProber(credentialsPath string, env []string) KeyringProber {
	return NewKeychainProber(DefaultSecurityBin, credentialsPath, env)
}
