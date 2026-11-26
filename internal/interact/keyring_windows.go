//go:build windows

package interact

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"unicode/utf16"
	"unsafe" //nolint:depguard // the Win32 CredReadW API takes raw credential structs

	"golang.org/x/sys/windows"
)

// The Windows credential store: the ZCode desktop keeps the 32-byte
// credential key as a generic credential in the Windows Credential Manager
// through @napi-rs/keyring (keyring-core + windows-native-keyring-store).
// The store maps (service, user) to a generic credential whose target name
// is "<user>.<service>" and whose blob is the UTF-16LE password. The user is
// sha256(path.resolve(credentials.json)) hex — the same derivation the Linux
// secret-tool lookup uses, so the wrapper reads the key the desktop wrote.
const (
	windowsCredentialTypeGeneric = 1

	// probeClassStoreUnavailable is the Windows-only probe class: the
	// credential exists in principle but the store refused the lookup.
	probeClassStoreUnavailable = "store-unavailable"

	// utf16UnitSize is the UTF-16 code-unit width in bytes.
	utf16UnitSize = 2
)

// ErrKeyringStoreUnavailable reports that the Windows Credential Manager
// refused the lookup (locked store, access denied, service failure).
var ErrKeyringStoreUnavailable = errors.New(
	"keyring probe: Windows Credential Manager is unavailable")

var (
	// errWindowsCredentialNotFound: no credential for the derived target —
	// the not-logged-in arm.
	errWindowsCredentialNotFound = errors.New("windows credential not found")

	// errWindowsCredentialShape: the credential exists but carries no usable
	// UTF-16LE blob.
	errWindowsCredentialShape = errors.New("windows credential has an unexpected shape")

	advapi32      = windows.NewLazySystemDLL("advapi32.dll") //nolint:gochecknoglobals // Win32 proc handle
	procCredReadW = advapi32.NewProc("CredReadW")            //nolint:gochecknoglobals // Win32 proc handle
	procCredFree  = advapi32.NewProc("CredFree")             //nolint:gochecknoglobals // Win32 proc handle
)

// windowsCredential mirrors CREDENTIALW (advapi32.h); the field order and
// alignment are load-bearing for the CredReadW out-parameter.
type windowsCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

// NewPlatformKeyLoader returns the Windows credential loader: the Credential
// Manager read of the key the desktop stored for this credentials.json path.
//
//nolint:ireturn // platform factory returns the shared KeyLoader interface
func NewPlatformKeyLoader(credentialsPath string, _ []string) KeyLoader {
	return windowsKeyLoader{credentialsPath: credentialsPath}
}

// NewPlatformKeyringProber returns the Windows proactive probe: the
// credentials file must exist and the Credential Manager must hold its key.
func NewPlatformKeyringProber(credentialsPath string, _ []string) KeyringProber {
	return func(ctx context.Context) error {
		started := time.Now()
		class, err := classifyWindowsKeyring(ctx, credentialsPath)

		slog.Info("interact: keyring probe", "class", class,
			"latency_ms", time.Since(started).Milliseconds())

		return err
	}
}

type windowsKeyLoader struct {
	credentialsPath string
}

// LoadKey reads and decodes the desktop's credential key.
func (l windowsKeyLoader) LoadKey(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("windows key loader: %w", err)
	}
	if l.credentialsPath == "" {
		return nil, errEmptyCredentialsPath
	}

	target, err := windowsCredentialTarget(l.credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoaderLookup, err)
	}

	password, err := readWindowsCredential(target)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoaderLookup, err)
	}

	key, err := decodeLoaderKey([]byte(password))
	if err != nil {
		return nil, err
	}

	return key, nil
}

// windowsCredentialTarget derives the Credential Manager target name the
// desktop's keyring store writes: sha256(path.resolve(credentialsPath)) hex,
// a ".", then the service name (windows-native-keyring-store's default
// delimiter; path.resolve does not resolve symlinks, so neither do we).
func windowsCredentialTarget(credentialsPath string) (string, error) {
	resolved, err := filepath.Abs(credentialsPath)
	if err != nil {
		return "", fmt.Errorf("resolve credentials path: %w", err)
	}

	sum := sha256.Sum256([]byte(resolved))

	return hex.EncodeToString(sum[:]) + "." + credentialKeyringService, nil
}

// readWindowsCredential reads the generic credential's UTF-16LE password.
func readWindowsCredential(target string) (string, error) {
	targetPtr, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return "", fmt.Errorf("encode credential target: %w", err)
	}

	var credential *windowsCredential

	ok, _, callErr := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetPtr)),
		windowsCredentialTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&credential)),
	)
	if ok == 0 {
		if errors.Is(callErr, windows.ERROR_NOT_FOUND) {
			return "", fmt.Errorf("%w: %w", errWindowsCredentialNotFound, callErr)
		}

		return "", fmt.Errorf("CredReadW: %w", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(credential))) //nolint:errcheck // CredFree returns void

	if credential.CredentialBlobSize == 0 || credential.CredentialBlob == nil {
		return "", errWindowsCredentialShape
	}

	blob := unsafe.Slice(credential.CredentialBlob, credential.CredentialBlobSize)

	return decodeWindowsCredentialBlob(blob)
}

// decodeWindowsCredentialBlob decodes the UTF-16LE password blob.
func decodeWindowsCredentialBlob(blob []byte) (string, error) {
	if len(blob)%2 != 0 {
		return "", errWindowsCredentialShape
	}

	units := make([]uint16, len(blob)/utf16UnitSize)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(blob[i*utf16UnitSize:])
	}

	return string(utf16.Decode(units)), nil
}

// classifyWindowsKeyring maps the lookup to the shared probe classes: a
// missing credentials file or a missing entry is the log-in hint; any other
// store failure is the unavailable arm.
func classifyWindowsKeyring(ctx context.Context, credentialsPath string) (string, error) {
	if credentialsPath == "" {
		return probeClassNoEntry, fmt.Errorf("%w: no credentials path", ErrKeyringNoEntry)
	}
	if _, err := os.Stat(credentialsPath); err != nil {
		return probeClassNoEntry, fmt.Errorf("%w: no readable desktop credentials (%w)", ErrKeyringNoEntry, err)
	}

	_, err := NewPlatformKeyLoader(credentialsPath, nil).LoadKey(ctx)

	switch {
	case err == nil:
		return probeClassOK, nil
	case errors.Is(err, errWindowsCredentialNotFound):
		return probeClassNoEntry,
			fmt.Errorf("%w: no Windows Credential Manager entry for this credentials file", ErrKeyringNoEntry)
	default:
		return probeClassStoreUnavailable, fmt.Errorf("%w: %w", ErrKeyringStoreUnavailable, err)
	}
}
