//go:build linux || windows

package interact

import "errors"

// errLoaderLookup is the shared platform loader failure sentinel. Darwin uses
// errSecurityLookup instead; keeping this definition out of the shared file
// avoids an unused-symbol verdict in the Darwin build.
var errLoaderLookup = errors.New("keyring lookup failed")
