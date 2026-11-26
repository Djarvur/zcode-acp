//go:build linux

package interact

// secretToolBin is the libsecret CLI the D-06 credential loader shells (a
// system binary on the guarded real tier — never a Go module, the T-5-SC
// zero-new-dependency posture).
const secretToolBin = "secret-tool"

// NewPlatformKeyLoader returns the Linux credential loader: the field-verified
// secret-tool lookup over the desktop's credentials path.
//
//nolint:ireturn // platform factory returns the shared KeyLoader interface
func NewPlatformKeyLoader(credentialsPath string, env []string) KeyLoader {
	return NewSecretToolKeyLoader(secretToolBin, credentialsPath, env)
}

// NewPlatformKeyringProber returns the Linux proactive probe: the secret-tool
// binary, its lookup contract, and the coded failure arms.
func NewPlatformKeyringProber(credentialsPath string, env []string) KeyringProber {
	return NewKeyringProber(secretToolBin, credentialsPath, env)
}
