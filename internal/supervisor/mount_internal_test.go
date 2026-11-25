//go:build linux

// In-package mount contract tests (plan 10-01 Task 2): helperEnv is the only
// helper-environment constructor and is unexported, so its exact allowlist is
// asserted here (the external test package cannot read it). The hostile-parent
// arm proves the allowlist posture BY CONSTRUCTION: exporting
// APPIMAGE_EXTRACT_AND_RUN, TARGET_APPIMAGE, APPDIR, APPIMAGE, NO_CLEANUP,
// VERBOSE, LD_PRELOAD, and LD_LIBRARY_PATH still yields only PATH/HOME/TMPDIR
// (T-10-01, RESEARCH Pitfall 5).

package supervisor

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
)

// mountEnvPathFallback is the documented PATH the helper receives when the
// parent carries none (the standard system binary dirs).
const mountEnvPathFallback = "/usr/local/bin:/usr/bin:/bin"

// errMountFailureUnrelated is the predicate test's non-sentinel error (err113:
// static anywhere it is compared).
var errMountFailureUnrelated = errors.New("unrelated")

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

// TestMountFailure pins the cancellation-vs-structural predicate the wrapper
// uses to decide whether a signal may mask a mount acquisition failure
// (CR-01): only a readiness error whose sole failure is the caller's
// cancellation is a clean shutdown; any structural sentinel — including a
// teardown error joined into a cancelled readiness error — must surface.
func TestMountFailure(t *testing.T) {
	readinessCancelled := fmt.Errorf("%w: %w", errMountReady, context.Canceled)

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errMountFailureUnrelated, false},
		{"pure readiness cancellation", readinessCancelled, false},
		{"readiness deadline", fmt.Errorf("%w: %w", errMountReady, context.DeadlineExceeded), false},
		{"readiness failure", fmt.Errorf("%w: helper exited", errMountReady), true},
		{"spawn failure", fmt.Errorf("%w: no such file", errMountSpawn), true},
		{"tree failure", fmt.Errorf("%w: missing zcode", errMountTree), true},
		{"base failure", fmt.Errorf("%w: unusable TMPDIR", errMountBase), true},
		{"install failure", fmt.Errorf("%w: not a packed AppImage", errMountInstall), true},
		{"teardown failure", fmt.Errorf("%w: helper not reaped", errMountTeardown), true},
		{"cancelled readiness joined with a failed teardown", errors.Join(readinessCancelled,
			fmt.Errorf("%w: mount still present", errMountTeardown)), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MountFailure(tc.err); got != tc.want {
				t.Fatalf("MountFailure(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
