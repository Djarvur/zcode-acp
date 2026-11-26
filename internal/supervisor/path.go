package supervisor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// loginShellTimeout bounds the optional PATH probe so a slow rc file can
	// never delay the backend spawn indefinitely.
	loginShellTimeout = 2 * time.Second

	// loginShellCap bounds the probe's stdout.
	loginShellCap = 64 << 10

	// loginShellPathVar is the one variable the probe replaces.
	loginShellPathVar = "PATH"

	// macLoginShellFallback is the login shell used when SHELL is absent or
	// names an unexpected binary.
	macLoginShellFallback = "/bin/zsh"
)

// loginShellRunner runs one bounded shell probe. It is a seam so tests never
// execute a real rc file.
type loginShellRunner func(ctx context.Context, shell string, args, env []string) ([]byte, error)

// EnrichParentPATH unions the caller PATH with the user's login-shell PATH on
// macOS. The backend inherits Orca's launchd-minimal PATH when Orca is started
// from the GUI, so without this union the model's tools (mise, go, node,
// package managers) disappear. The probe is non-interactive (`-lc`, never
// `-i`), bounded, and only PATH crosses into ChildEnv; the child env builder
// still scrubs proxies and every non-allowlisted variable.
//
// Non-Darwin hosts are returned unchanged so Linux behavior is untouched.
func EnrichParentPATH(ctx context.Context, env []string) []string {
	if runtime.GOOS != goosDarwin {
		return env
	}

	return enrichParentPATH(ctx, env, defaultLoginShellRunner)
}

// enrichParentPATH is the platform-neutral implementation behind the exported
// Darwin gate.
func enrichParentPATH(ctx context.Context, env []string, run loginShellRunner) []string {
	callerPath := envSliceValue(env, loginShellPathVar)
	shell := safeLoginShell(envSliceValue(env, "SHELL"))

	out, err := run(ctx, shell, loginShellArgs(shell), loginShellEnv(env))
	if err != nil {
		slog.Debug("supervisor: login-shell PATH probe failed; keeping caller PATH", "err", err)

		return env
	}

	shellPath := lastNonEmptyLine(string(out))
	if shellPath == "" {
		return env
	}

	union := unionPATH(shellPath, callerPath)
	if union == callerPath {
		return env
	}

	return setEnv(env, loginShellPathVar, union)
}

// safeLoginShell restricts SHELL to the shells the probe supports and falls
// back to the macOS default.
func safeLoginShell(shell string) string {
	switch filepath.Base(shell) {
	case "zsh", "bash", "sh":
		return shell
	default:
		return macLoginShellFallback
	}
}

// loginShellArgs builds the non-interactive login-shell probe: `-lc` reads the
// login files, and zsh/bash additionally source the interactive rc (toolchain
// paths commonly live there) with both streams discarded so rc noise can never
// corrupt the single PATH line.
func loginShellArgs(shell string) []string {
	var rc string

	switch filepath.Base(shell) {
	case "zsh":
		rc = `[ -r "$HOME/.zshrc" ] && . "$HOME/.zshrc" >/dev/null 2>&1; `
	case "bash":
		rc = `[ -r "$HOME/.bashrc" ] && . "$HOME/.bashrc" >/dev/null 2>&1; `
	}

	script := rc + `printf '%s\n' "$PATH"`

	return []string{"-lc", script}
}

// loginShellEnv is the probe's environment allowlist. It deliberately excludes
// proxy variables and arbitrary secrets: only what a login rc needs to resolve
// a toolchain crosses.
func loginShellEnv(env []string) []string {
	allow := map[string]struct{}{
		"PATH": {}, "HOME": {}, "SHELL": {}, "USER": {}, "LOGNAME": {},
		"LANG": {}, "LC_ALL": {},
	}
	out := make([]string, 0, len(allow))

	for _, kv := range env {
		key, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if _, allowed := allow[key]; allowed {
			out = append(out, kv)
		}
	}

	return out
}

// cappedWriter is a bounded byte sink: bytes beyond limit are dropped while
// Write still claims the whole chunk, so a chatty rc can never grow memory.
type cappedWriter struct {
	buf   bytes.Buffer
	limit int
}

// Write implements io.Writer over the capped sink.
func (w *cappedWriter) Write(p []byte) (int, error) {
	n := len(p)
	if room := w.limit - w.buf.Len(); room > 0 {
		if len(p) > room {
			p = p[:room]
		}
		_, _ = w.buf.Write(p)
	}

	return n, nil
}

// String returns the captured bytes.
func (w *cappedWriter) String() string { return w.buf.String() }

// defaultLoginShellRunner runs the probe under the bounded timeout and cap.
func defaultLoginShellRunner(ctx context.Context, shell string, args, env []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, loginShellTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = env

	var out cappedWriter

	out.limit = loginShellCap

	cmd.Stdout = &out
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run login shell %s: %w", shell, err)
	}

	return []byte(out.String()), nil
}

// lastNonEmptyLine returns the last non-empty trimmed line (rc noise may
// precede the PATH line; the probe itself writes it last).
func lastNonEmptyLine(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}

	return ""
}

// unionPATH merges shell entries first, then caller entries, deduplicated.
func unionPATH(shellPath, callerPath string) string {
	seen := make(map[string]struct{})
	entries := make([]string, 0, len(filepath.SplitList(shellPath))+len(filepath.SplitList(callerPath)))

	for _, list := range []string{shellPath, callerPath} {
		for _, entry := range filepath.SplitList(list) {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			if _, ok := seen[entry]; ok {
				continue
			}
			seen[entry] = struct{}{}
			entries = append(entries, entry)
		}
	}

	return strings.Join(entries, string(os.PathListSeparator))
}

// envValue reads one variable from an env slice.
func envSliceValue(env []string, key string) string {
	prefix := key + "="
	for _, kv := range env {
		if value, ok := strings.CutPrefix(kv, prefix); ok {
			return value
		}
	}

	return ""
}

// setEnv returns a copy of env with key set, preserving the original order.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	replaced := false

	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			if !replaced {
				out = append(out, prefix+value)
				replaced = true
			}

			continue
		}
		out = append(out, kv)
	}
	if !replaced {
		out = append(out, prefix+value)
	}

	return out
}

// FindExecutable resolves name through an explicit PATH value without
// consulting the process environment (exec.LookPath cannot take one).
func FindExecutable(name, pathValue string) string {
	for _, dir := range filepath.SplitList(pathValue) {
		if dir == "" {
			continue
		}

		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate
		}
	}

	return ""
}
