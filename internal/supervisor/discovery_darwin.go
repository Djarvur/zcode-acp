//go:build darwin

package supervisor

import "context"

// platformBinaryName is the macOS app binary name; the Darwin port will
// refine the bundle layout (ZCode.app/Contents/...).
func platformBinaryName() string { return "ZCode" }

// probeInstallStage is the macOS install-layout stage. Native macOS
// discovery is not implemented yet: the wrapper resolves the app through the
// explicit --app-server/ZCODE_ACP_APP_SERVER override.
func (Probers) probeInstallStage(context.Context, bool) ([]Probe, Install, bool) {
	row := Probe{Name: "darwin-install", Query: "/Applications/ZCode.app"}
	row.Result = "native macOS discovery is not implemented yet; " +
		"use --app-server/ZCODE_ACP_APP_SERVER"

	return []Probe{row}, Install{}, false
}

// DefaultProbers returns the Darwin seam set: only the override is wired.
func DefaultProbers(home, optDir string) Probers {
	return Probers{Override: defaultOverride, Home: home, OptDir: optDir}
}

// builtinProviderRelForPlatform reports that the macOS runtime-provider cache
// is not probed yet; the bundled copy fills the builtin env.
func builtinProviderRelForPlatform() (string, bool) {
	return "", false
}

// DefaultOptDir is empty on macOS: the Linux package layout
// (/opt/ZCode) does not exist here.
func DefaultOptDir() string { return "" }
