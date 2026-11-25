package interact

import "runtime"

// NewDesktopKeyLoader returns the platform's host-side key reader: macOS
// Keychain on Darwin, Secret Service elsewhere.
//
//nolint:ireturn // the factory intentionally returns the KeyLoader seam
func NewDesktopKeyLoader(credentialsPath string, env []string) KeyLoader {
	if runtime.GOOS == goosDarwin {
		return NewSecurityKeyLoader(DefaultSecurityBin, credentialsPath, env)
	}

	return NewSecretToolKeyLoader(DefaultSecretToolBin, credentialsPath, env)
}

// NewDesktopKeyringProber returns the proactive probe for the host platform.
func NewDesktopKeyringProber(credentialsPath string, env []string) KeyringProber {
	if runtime.GOOS == goosDarwin {
		return NewKeychainProber(DefaultSecurityBin, credentialsPath, env)
	}

	return NewKeyringProber(DefaultSecretToolBin, credentialsPath, env)
}

const goosDarwin = "darwin"
