package supervisor_test

import (
	"testing"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

const (
	testDarwin = "darwin"
	testLinux  = "linux"
	testArm64  = "arm64"
	testAMD64  = "amd64"
)

func TestProviderPlatformDir(t *testing.T) {
	cases := []struct {
		goos   string
		goarch string
		want   string
	}{
		{testDarwin, testArm64, "darwin-aarch64"},
		{testDarwin, testAMD64, "darwin-x86_64"},
		{testLinux, testAMD64, "linux-x86_64"},
		{testLinux, testArm64, "linux-aarch64"},
		{"windows", testAMD64, ""},
		{testDarwin, "386", ""},
	}

	for _, tc := range cases {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			if got := supervisor.ProviderPlatformDir(tc.goos, tc.goarch); got != tc.want {
				t.Fatalf("ProviderPlatformDir(%q, %q) = %q, want %q",
					tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}
