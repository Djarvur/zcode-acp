package supervisor

// windowsBinaryName is the Windows Electron binary; the native install stage
// validates <root>\ZCode.exe beside the layout-invariant entrypoint.
const windowsBinaryName = "ZCode.exe"

// Windows install-stage layout labels.
const (
	layoutWindowsUser     = "windows-user"
	layoutWindowsMachine  = "windows-machine"
	layoutWindowsRegistry = "windows-registry"
)

func windowsBundleInstall(string, string) (Install, bool) { return Install{}, false }

func windowsInstallStage(WindowsSeams, bool) ([]Probe, Install, bool) {
	return nil, Install{}, false
}

func windowsRegistryStage(func() ([]UninstallEntry, error)) ([]Probe, Install, bool) {
	return nil, Install{}, false
}

func windowsInstallRoot(UninstallEntry) string { return "" }

func windowsDisplayIconPath(string) string { return "" }

func windowsDisplayNameMatches(string) bool { return false }
