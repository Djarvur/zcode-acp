//go:build windows

package interact

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

// TestWindowsCredentialTarget pins the Windows Credential Manager target the
// desktop's keyring store writes: sha256(path.resolve(credentials.json)) hex,
// a ".", then the service name. The hash is the exact value the live 4.0.0
// install produced for C:\Users\zcode\.zcode\v2\credentials.json.
func TestWindowsCredentialTarget(t *testing.T) {
	const (
		path       = `C:\Users\zcode\.zcode\v2\credentials.json`
		wantTarget = "45d6cd1f21c945d45b2e80b5e68b05499ea8af78cf03b72bd2ab38af64c6fe16.ZCode Credential Encryption"
	)

	got, err := windowsCredentialTarget(path)
	if err != nil {
		t.Fatalf("windowsCredentialTarget: %v", err)
	}

	if got != wantTarget {
		t.Fatalf("target = %q, want %q", got, wantTarget)
	}
}

// TestDecodeWindowsCredentialBlob pins the UTF-16LE blob decode the
// windows-native-keyring-store uses for the keyring password.
func TestDecodeWindowsCredentialBlob(t *testing.T) {
	const password = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	units := utf16.Encode([]rune(password))
	blob := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(blob[i*2:], unit)
	}

	got, err := decodeWindowsCredentialBlob(blob)
	if err != nil {
		t.Fatalf("decodeWindowsCredentialBlob: %v", err)
	}

	if got != password {
		t.Fatalf("decoded password = %q, want %q", got, password)
	}
}
