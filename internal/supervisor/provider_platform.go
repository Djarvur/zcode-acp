package supervisor

const (
	goosDarwin  = "darwin"
	goosLinux   = "linux"
	goarchArm64 = "arm64"
	goarchAMD64 = "amd64"
)

// ProviderPlatformDir maps a Go GOOS/GOARCH pair to the ZCode runtime-provider
// cache platform directory used by the desktop build. The observed macOS
// layout is darwin-aarch64; Linux uses linux-x86_64. An unknown pair returns
// an empty string and discovery reports an honest miss.
func ProviderPlatformDir(goos, goarch string) string {
	switch goos {
	case goosDarwin:
		switch goarch {
		case goarchArm64:
			return "darwin-aarch64"
		case goarchAMD64:
			return "darwin-x86_64"
		}
	case goosLinux:
		switch goarch {
		case goarchAMD64:
			return "linux-x86_64"
		case goarchArm64:
			return "linux-aarch64"
		}
	}

	return ""
}
