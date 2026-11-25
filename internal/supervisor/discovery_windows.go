//go:build windows

package supervisor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// Uninstall-registry facts: every installed product writes one subkey under
// this key (HKCU and both HKLM views), and ReadSubKeyNames(-1) asks for every
// subkey name.
const (
	uninstallKeyRel    = `Software\Microsoft\Windows\CurrentVersion\Uninstall`
	uninstallSubKeyAll = -1
)

// The uninstall value names the discovery model reads; absent values and
// non-SZ values are the empty string.
const (
	valueDisplayName     = "DisplayName"
	valueDisplayVersion  = "DisplayVersion"
	valueInstallLocation = "InstallLocation"
	valueDisplayIcon     = "DisplayIcon"
)

// platformBinaryName is the Windows Electron binary name.
func platformBinaryName() string {
	return windowsBinaryName
}

// probeInstallStage delegates to the platform-neutral Windows stage: the
// environment-derived directory sources and the uninstall-registry records.
func (p Probers) probeInstallStage(_ context.Context, found bool) ([]Probe, Install, bool) {
	return windowsInstallStage(p.Windows, found)
}

// DefaultProbers returns the Windows seam set: the override plus native
// install discovery — environment-derived directory sources and the read-only
// uninstall-registry enumeration. The Linux packaging probes do not exist
// here.
func DefaultProbers(home, optDir string) Probers {
	return Probers{
		Override: defaultOverride,
		Home:     home,
		OptDir:   optDir,
		Windows: WindowsSeams{
			Dirs:      defaultWindowsDirs(),
			Uninstall: defaultUninstallEntries,
		},
	}
}

// defaultWindowsDirs builds the native Windows install sources in probe order:
// the per-user tree first, then the per-machine Program Files trees (the
// x86 one only when the variable is set and names a different tree). An unset
// variable leaves the source Root empty so the stage records an honest
// "not defined on this host" row.
func defaultWindowsDirs() []WindowsInstallDir {
	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")

	dirs := []WindowsInstallDir{
		{
			Name:   "windows-localappdata",
			Query:  `%LOCALAPPDATA%\Programs\ZCode`,
			Layout: layoutWindowsUser,
		},
		{
			Name:   "windows-programfiles",
			Query:  `%ProgramFiles%\ZCode`,
			Layout: layoutWindowsMachine,
		},
	}

	if localAppData != "" {
		dirs[0].Root = filepath.Join(localAppData, "Programs", "ZCode")
	}

	if programFiles != "" {
		dirs[1].Root = filepath.Join(programFiles, "ZCode")
	}

	if programFilesX86 == "" {
		return dirs
	}

	x86Root := filepath.Join(programFilesX86, "ZCode")
	if x86Root == dirs[1].Root {
		return dirs
	}

	return append(dirs, WindowsInstallDir{
		Name:   "windows-programfiles-x86",
		Query:  `%ProgramFiles(x86)%\ZCode`,
		Layout: layoutWindowsMachine,
		Root:   x86Root,
	})
}

// uninstallHive names one registry source of uninstall records: the hive
// handle, its diagnostics label, and the registry view to open.
type uninstallHive struct {
	hive  registry.Key
	label string
	view  uint32
}

// uninstallHives returns the fixed enumeration order: HKCU, then HKLM's
// 64-bit view, then HKLM's 32-bit view.
func uninstallHives() []uninstallHive {
	return []uninstallHive{
		{hive: registry.CURRENT_USER, label: "HKCU", view: 0},
		{hive: registry.LOCAL_MACHINE, label: "HKLM", view: registry.WOW64_64KEY},
		{hive: registry.LOCAL_MACHINE, label: "HKLM", view: registry.WOW64_32KEY},
	}
}

// defaultUninstallEntries reads the uninstall-registry records through
// golang.org/x/sys/windows/registry in read-only mode. Strings only become
// candidate roots: nothing here is executed, PATH-resolved, or
// shell-interpreted. A partially unreadable registry returns every
// successfully read entry plus one wrapped joined error, so a locked HKCU
// never hides HKLM results.
func defaultUninstallEntries() ([]UninstallEntry, error) {
	entries := make([]UninstallEntry, 0)
	errs := make([]error, 0)

	for _, source := range uninstallHives() {
		found, err := readUninstallHive(source)
		entries = append(entries, found...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return entries, fmt.Errorf("read the uninstall registry: %w", errors.Join(errs...))
	}

	return entries, nil
}

// readUninstallHive enumerates one hive's uninstall subkeys and reads the four
// record values. Subkeys that cannot be read are collected and skipped; the
// readable records are still returned.
func readUninstallHive(source uninstallHive) ([]UninstallEntry, error) {
	keyPath := source.label + `\` + uninstallKeyRel
	access := uint32(registry.QUERY_VALUE | registry.ENUMERATE_SUB_KEYS | source.view)

	key, err := registry.OpenKey(source.hive, uninstallKeyRel, access)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", keyPath, err)
	}
	defer func() { _ = key.Close() }()

	names, err := key.ReadSubKeyNames(uninstallSubKeyAll)
	if err != nil {
		return nil, fmt.Errorf("enumerate %s: %w", keyPath, err)
	}

	entries := make([]UninstallEntry, 0, len(names))
	errs := make([]error, 0)

	for _, name := range names {
		entry, readErr := readUninstallEntry(key, keyPath, name, source.view)
		if readErr != nil {
			errs = append(errs, readErr)

			continue
		}

		// Legacy uninstall keys can carry no name, path, or icon at all
		// (AddressBook, Connection Manager, …). They have no discovery
		// value, so the enumeration's contract exposes only usable records.
		if entry.DisplayName == "" && entry.InstallLocation == "" && entry.DisplayIcon == "" {
			continue
		}

		entries = append(entries, entry)
	}

	if len(errs) > 0 {
		return entries, fmt.Errorf("read %s: %w", keyPath, errors.Join(errs...))
	}

	return entries, nil
}

// readUninstallEntry reads one uninstall subkey's four record values. The Key
// is the hive label plus the fixed key path plus the subkey name.
func readUninstallEntry(parent registry.Key, parentPath, name string, view uint32) (UninstallEntry, error) {
	keyPath := parentPath + `\` + name
	access := uint32(registry.QUERY_VALUE | view)

	key, err := registry.OpenKey(parent, name, access)
	if err != nil {
		return UninstallEntry{}, fmt.Errorf("open %s: %w", keyPath, err)
	}
	defer func() { _ = key.Close() }()

	entry := UninstallEntry{Key: keyPath}
	values := []struct {
		name string
		dst  *string
	}{
		{valueDisplayName, &entry.DisplayName},
		{valueDisplayVersion, &entry.DisplayVersion},
		{valueInstallLocation, &entry.InstallLocation},
		{valueDisplayIcon, &entry.DisplayIcon},
	}

	errs := make([]error, 0)

	for _, value := range values {
		read, readErr := registryString(key, value.name)
		if readErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", keyPath, readErr))

			continue
		}

		*value.dst = read
	}

	if len(errs) > 0 {
		return UninstallEntry{}, errors.Join(errs...)
	}

	return entry, nil
}

// registryString reads one string value; an absent value and a non-SZ value
// are the empty string, so optional record values never fail a record.
func registryString(key registry.Key, name string) (string, error) {
	value, _, err := key.GetStringValue(name)
	if err == nil {
		return value, nil
	}

	if errors.Is(err, registry.ErrNotExist) || errors.Is(err, registry.ErrUnexpectedType) {
		return "", nil
	}

	return "", fmt.Errorf("read %s: %w", name, err)
}

// builtinProviderRelForPlatform reports that the Windows runtime-provider
// cache is not probed yet; the bundled copy fills the builtin env.
func builtinProviderRelForPlatform() (string, bool) {
	return "", false
}

// DefaultOptDir is empty on Windows: the Linux package layout
// (/opt/ZCode) does not exist here.
func DefaultOptDir() string { return "" }
