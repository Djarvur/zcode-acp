//go:build !windows

package supervisor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	fixtureHomeEnv   = "HOME=/home/fixture"
	fixtureZshEnv    = "SHELL=/bin/zsh"
	fixtureCallerEnv = "PATH=/caller/bin"
)

var errFakeShellRun = errors.New("shell failed")

const (
	loginShellHelperModeEnv    = "ZCODE_ACP_TEST_LOGIN_SHELL_HELPER"
	loginShellHelperParent     = "parent"
	loginShellHelperDescendant = "descendant"
	loginShellHelperPIDEnv     = "ZCODE_ACP_TEST_LOGIN_SHELL_HELPER_PID"
	loginShellDescendantPIDEnv = "ZCODE_ACP_TEST_LOGIN_SHELL_DESCENDANT_PID"

	testLoginShellWaitDelay      = 100 * time.Millisecond
	testLoginShellContextTimeout = 500 * time.Millisecond
	testLoginShellWatchdog       = 2 * time.Second
	testLoginShellDescendantLife = 5 * time.Second
)

// TestLoginShellDescendantHelper is a Go re-exec fixture. The parent starts a
// same-group descendant with inherited stdout, records both PIDs, and exits;
// the descendant holds that stdout open for a bounded lifetime.
func TestLoginShellDescendantHelper(t *testing.T) {
	switch os.Getenv(loginShellHelperModeEnv) {
	case loginShellHelperParent:
		writeLoginShellPID(t, os.Getenv(loginShellHelperPIDEnv), os.Getpid())
		descendant := exec.Command(os.Args[0], "-test.run=^TestLoginShellDescendantHelper$")
		descendant.Env = append(os.Environ(), loginShellHelperModeEnv+"="+loginShellHelperDescendant)
		descendant.Stdin = nil
		descendant.Stdout = os.Stdout
		descendant.Stderr = os.Stderr
		if err := descendant.Start(); err != nil {
			t.Fatalf("start descendant: %v", err)
		}
		writeLoginShellPID(t, os.Getenv(loginShellDescendantPIDEnv), descendant.Process.Pid)
	case loginShellHelperDescendant:
		time.Sleep(testLoginShellDescendantLife)
	}
}

func writeLoginShellPID(t *testing.T, path string, pid int) {
	t.Helper()
	if path == "" || pid <= 0 {
		t.Fatalf("invalid PID record path=%q pid=%d", path, pid)
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		t.Fatalf("write PID record %s: %v", path, err)
	}
}

func readLoginShellPID(t *testing.T, path string, required bool) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) && !required {
		return 0
	}
	if err != nil {
		t.Fatalf("read PID record %s: %v", path, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("invalid PID record %s=%q: %v", path, string(data), err)
	}
	return pid
}

// runLoginShellTimed keeps the test's millisecond WaitDelay explicit. During
// RED it delegates to the current production runner, which has no such seam.
func runLoginShellTimed(
	ctx context.Context, shell string, args, env []string, waitDelay time.Duration,
) ([]byte, error) {
	_ = waitDelay
	return defaultLoginShellRunner(ctx, shell, args, env)
}

type loginShellResult struct {
	out []byte
	err error
}

// TestLoginShellRunnerDescendantStdout proves the production probe cannot hang
// on exec's stdout copier after the direct shell exits. The same-group
// descendant must be killed and the group observed gone before return.
func TestLoginShellRunnerDescendantStdout(t *testing.T) {
	fixture := t.TempDir()
	helperPIDFile := filepath.Join(fixture, "helper.pid")
	descendantPIDFile := filepath.Join(fixture, "descendant.pid")

	t.Cleanup(func() {
		helperPID := readLoginShellPID(t, helperPIDFile, false)
		descendantPID := readLoginShellPID(t, descendantPIDFile, false)
		if helperPID > 0 {
			if err := syscall.Kill(-helperPID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Errorf("kill helper group %d: %v", helperPID, err)
			}
		}
		if descendantPID > 0 {
			if err := syscall.Kill(descendantPID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Errorf("kill descendant %d: %v", descendantPID, err)
			}
		}
	})

	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		loginShellHelperModeEnv + "=" + loginShellHelperParent,
		loginShellHelperPIDEnv + "=" + helperPIDFile,
		loginShellDescendantPIDEnv + "=" + descendantPIDFile,
	}
	ctx, cancel := context.WithTimeout(context.Background(), testLoginShellContextTimeout)
	defer cancel()

	resultCh := make(chan loginShellResult, 1)
	start := time.Now()
	go func() {
		out, err := runLoginShellTimed(ctx, os.Args[0],
			[]string{"-test.run=^TestLoginShellDescendantHelper$"}, env, testLoginShellWaitDelay)
		resultCh <- loginShellResult{out: out, err: err}
	}()

	select {
	case result := <-resultCh:
		if result.err == nil {
			t.Fatal("probe returned success while its descendant still held stdout")
		}
		if elapsed := time.Since(start); elapsed > testLoginShellWatchdog {
			t.Fatalf("probe returned after %v, want the %v watchdog bound", elapsed, testLoginShellWatchdog)
		}
	case <-time.After(testLoginShellWatchdog):
		t.Fatal("probe hung after the direct helper exited and its descendant inherited stdout")
	}

	helperPID := readLoginShellPID(t, helperPIDFile, true)
	descendantPID := readLoginShellPID(t, descendantPIDFile, true)
	assertLoginShellGone(t, -helperPID, "helper process group")
	assertLoginShellGone(t, descendantPID, "descendant process")
}

func assertLoginShellGone(t *testing.T, pid int, what string) {
	t.Helper()
	deadline := time.Now().Add(testLoginShellWatchdog)
	for {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s (%d) still exists after the bounded probe", what, pid)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// fakeLoginRunner captures one probe invocation and returns canned output.
type fakeLoginRunner struct {
	out   []byte
	err   error
	shell string
	args  []string
	env   []string
}

func (f *fakeLoginRunner) run(_ context.Context, shell string, args, env []string) ([]byte, error) {
	f.shell = shell
	f.args = slices.Clone(args)
	f.env = slices.Clone(env)

	return f.out, f.err
}

func TestEnrichParentPATHUnion(t *testing.T) {
	runner := &fakeLoginRunner{out: []byte("/shell/bin:/caller/bin:/usr/bin\n")}
	env := []string{
		"PATH=/caller/bin:/usr/bin:/opt/local/bin",
		fixtureHomeEnv,
		fixtureZshEnv,
		"LANG=en_US.UTF-8",
		"HTTP_PROXY=http://192.0.2.1:9",
		"SECRET=must-not-reach-the-probe",
	}

	got := enrichParentPATH(context.Background(), env, runner.run)

	want := "/shell/bin:/caller/bin:/usr/bin:/opt/local/bin"
	if path := envSliceValue(got, "PATH"); path != want {
		t.Fatalf("PATH = %q, want the shell-first union %q", path, want)
	}
	if runner.shell != "/bin/zsh" {
		t.Fatalf("probe shell = %q, want /bin/zsh", runner.shell)
	}
	if len(runner.args) != 2 || runner.args[0] != "-lc" {
		t.Fatalf("probe args = %v, want non-interactive -lc", runner.args)
	}
	if !strings.Contains(runner.args[1], ".zshrc") {
		t.Fatalf("zsh probe must source .zshrc: %q", runner.args[1])
	}

	for _, key := range []string{"PATH", "HOME", "SHELL", "LANG"} {
		if envSliceValue(runner.env, key) == "" {
			t.Fatalf("probe env missing %s: %v", key, runner.env)
		}
	}
	for _, key := range []string{"HTTP_PROXY", "SECRET"} {
		if envSliceValue(runner.env, key) != "" {
			t.Fatalf("probe env leaked %s: %v", key, runner.env)
		}
	}
}

func TestEnrichParentPATHFallbacks(t *testing.T) {
	t.Run("probe failure keeps caller PATH", func(t *testing.T) {
		runner := &fakeLoginRunner{err: errFakeShellRun}
		env := []string{fixtureCallerEnv, fixtureHomeEnv, fixtureZshEnv}

		got := enrichParentPATH(context.Background(), env, runner.run)
		if path := envSliceValue(got, "PATH"); path != "/caller/bin" {
			t.Fatalf("PATH = %q, want the caller PATH after a probe failure", path)
		}
	})

	t.Run("empty probe output keeps caller PATH", func(t *testing.T) {
		runner := &fakeLoginRunner{out: []byte("\n")}
		env := []string{fixtureCallerEnv, fixtureHomeEnv, fixtureZshEnv}

		got := enrichParentPATH(context.Background(), env, runner.run)
		if path := envSliceValue(got, "PATH"); path != "/caller/bin" {
			t.Fatalf("PATH = %q, want the caller PATH after empty output", path)
		}
	})

	t.Run("unsafe SHELL falls back to zsh", func(t *testing.T) {
		runner := &fakeLoginRunner{out: []byte("/shell/bin\n")}
		env := []string{fixtureCallerEnv, fixtureHomeEnv, "SHELL=/tmp/evil"}

		enrichParentPATH(context.Background(), env, runner.run)
		if runner.shell != "/bin/zsh" {
			t.Fatalf("probe shell = %q, want the safe fallback", runner.shell)
		}
	})

	t.Run("bash sources bashrc", func(t *testing.T) {
		runner := &fakeLoginRunner{out: []byte("/shell/bin\n")}
		env := []string{fixtureCallerEnv, fixtureHomeEnv, "SHELL=/bin/bash"}

		enrichParentPATH(context.Background(), env, runner.run)
		if !strings.Contains(runner.args[1], ".bashrc") {
			t.Fatalf("bash probe must source .bashrc: %q", runner.args[1])
		}
	})
}

func TestUnionPATH(t *testing.T) {
	got := unionPATH("/a:/b:/a", "/b:/c::/d")
	want := "/a:/b:/c:/d"
	if got != want {
		t.Fatalf("unionPATH = %q, want %q", got, want)
	}
}

func TestLastNonEmptyLine(t *testing.T) {
	got := lastNonEmptyLine("rc noise\n\n/shell/bin:/usr/bin\n")
	if got != "/shell/bin:/usr/bin" {
		t.Fatalf("lastNonEmptyLine = %q", got)
	}
}
