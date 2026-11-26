package supervisor

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

const (
	fixtureHomeEnv   = "HOME=/home/fixture"
	fixtureZshEnv    = "SHELL=/bin/zsh"
	fixtureCallerEnv = "PATH=/caller/bin"
)

var errFakeShellRun = errors.New("shell failed")

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
