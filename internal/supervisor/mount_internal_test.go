// In-package mount contract tests (plan 10-01 Task 2): helperEnv is the only
// helper-environment constructor and is unexported, so its exact allowlist is
// asserted here (the external test package cannot read it). The hostile-parent
// arm proves the allowlist posture BY CONSTRUCTION: exporting
// APPIMAGE_EXTRACT_AND_RUN, TARGET_APPIMAGE, APPDIR, APPIMAGE, NO_CLEANUP,
// VERBOSE, LD_PRELOAD, and LD_LIBRARY_PATH still yields only PATH/HOME/TMPDIR
// (T-10-01, RESEARCH Pitfall 5).

package supervisor

import (
	"slices"
	"testing"
)

// mountEnvPathFallback is the documented PATH the helper receives when the
// parent carries none (the standard system binary dirs).
const mountEnvPathFallback = "/usr/local/bin:/usr/bin:/bin"

// TestMountEnv pins the helper environment: exactly the allowlist, with the
// documented PATH fallback when the parent carries none.
func TestMountEnv(t *testing.T) {
	t.Run("allowlist by construction", testMountEnvAllowlist)
	t.Run("PATH fallback when the parent has none", testMountEnvPathFallback)
}

// testMountEnvAllowlist proves a hostile parent environment produces exactly
// the three allowlisted variables.
func testMountEnvAllowlist(t *testing.T) {
	t.Setenv(allowlistPathVar, "/host/bin:/usr/bin")
	t.Setenv(allowlistHomeVar, "/host/home")

	for _, key := range []string{
		"APPIMAGE_EXTRACT_AND_RUN",
		"TARGET_APPIMAGE",
		"APPDIR",
		"APPIMAGE",
		"NO_CLEANUP",
		"VERBOSE",
		"LD_PRELOAD",
		"LD_LIBRARY_PATH",
	} {
		t.Setenv(key, "hostile")
	}

	got := helperEnv("/tmp/zcode-acp-mount-fixture")
	slices.Sort(got)

	want := []string{
		allowlistHomeVar + "=/host/home",
		allowlistPathVar + "=/host/bin:/usr/bin",
		mountTmpdirVar + "=/tmp/zcode-acp-mount-fixture",
	}
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Fatalf("helperEnv = %v, want exactly the allowlist %v "+
			"(APPIMAGE_EXTRACT_AND_RUN and the loader vars must never cross)", got, want)
	}
}

// testMountEnvPathFallback proves the documented /usr/local/bin fallback when
// the parent carries no PATH (a bare env would otherwise execute the helper
// runtime without a usable PATH).
func testMountEnvPathFallback(t *testing.T) {
	t.Setenv(allowlistPathVar, "")
	t.Setenv(allowlistHomeVar, "/host/home")

	got := helperEnv("/tmp/zcode-acp-mount-fixture")

	if !slices.Contains(got, allowlistPathVar+"="+mountEnvPathFallback) {
		t.Fatalf("helperEnv = %v, want PATH=%s when the parent has none", got, mountEnvPathFallback)
	}

	if len(got) != 3 {
		t.Fatalf("helperEnv = %v, want the 3-variable allowlist", got)
	}
}
