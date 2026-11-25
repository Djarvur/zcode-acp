//go:build windows

package supervisor

import "context"

func platformBinaryName() string {
	return "ZCode.exe"
}

// probeInstallStage is the Windows install-layout stage. Native Windows
// discovery is not implemented yet: the wrapper resolves an installed tree
// through the explicit --app-server/ZCODE_ACP_APP_SERVER override.
func (Probers) probeInstallStage(context.Context, bool) ([]Probe, Install, bool) {
	row := Probe{
		Name:  "windows-install",
		Query: "%LOCALAPPDATA%\\Programs\\ZCode\\ZCode.exe",
	}
	row.Result = "native Windows discovery is not implemented yet; " +
		"use --app-server/ZCODE_ACP_APP_SERVER with the installed ZCode.exe"

	return []Probe{row}, Install{}, false
}

// DefaultProbers returns the Windows seam set: only the override is wired;
// the Linux packaging probes do not exist here.
func DefaultProbers(home, optDir string) Probers {
	return Probers{Override: defaultOverride, Home: home, OptDir: optDir}
}

// builtinProviderRelForPlatform reports that the Windows runtime-provider
// cache is not probed yet; the bundled copy fills the builtin env.
func builtinProviderRelForPlatform() (string, bool) {
	return "", false
}

// DefaultOptDir is empty on Windows: the Linux package layout
// (/opt/ZCode) does not exist here.
func DefaultOptDir() string { return "" }
