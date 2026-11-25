// Spawn contract (BRIDGE-03 part 2, plan 03-07): the canonical argv line and
// child environment from docs/protocol/install-discovery.md :48-52 (the
// ELECTRON_RUN_AS_NODE spawn line, consumed verbatim), plus SpawnChild — the
// process-group start (Setpgid, so -PgID addresses the whole tree incl.
// grandchildren) with all three stdio pipes wired, the child's stderr drained
// capped (PITFALL 2: an undrained 64 KB pipe buffer stalls the child
// mid-turn), and an exit monitor that reaps exactly once. The env is an
// explicit allowlist with the proxy variables scrubbed (D-03 / PITFALL 9 /
// T-3-08): Go caches ProxyFromEnvironment once, and a dead or hostile parent
// proxy var must never redirect child traffic.

package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Spawn-contract facts (docs/protocol/install-discovery.md :48-52).
const (
	// childRunAsNodeVar/Value make the Electron binary run as plain Node.
	childRunAsNodeVar   = "ELECTRON_RUN_AS_NODE"
	childRunAsNodeValue = "1"
	// childBuiltinVar carries the newest zcode-builtin.json into the child.
	childBuiltinVar = "ZCODE_BUILTIN_PROVIDER_CONFIG_FILE"
	// childPersonalVar carries the personal provider config into the child:
	// resolveNodeProviderRuntimePaths requires the builtin and personal paths
	// PAIRED — a worker started with only the builtin path runs its provider
	// registry degraded (updateAccountConfig acks, account models never
	// publish — EXP-CODINGPLAN-CYCLE-01 attempts 1-6). The desktop passes the
	// same pair (provider-runtime-env.ts:88-99).
	childPersonalVar = "ZCODE_PERSONAL_PROVIDER_CONFIG_FILE"
	// appServerSubcommand and stdioFlag are the argv tail of the spawn line.
	appServerSubcommand = "app-server"
	stdioFlag           = "--stdio"
)

// errEmptyArgv is SpawnChild's contract-violation sentinel.
var errEmptyArgv = errors.New("zcode: spawn: empty argv")

// Env-contract facts (D-03 / PITFALL 9 / T-3-08).
const (
	// allowlistPathVar and allowlistHomeVar are the only parent vars copied
	// into the child. The proxy variables (HTTP_PROXY/HTTPS_PROXY/ALL_PROXY
	// in both cases — Linux getenv is case-sensitive and Node reads the
	// lowercase forms) are scrubbed by construction: the allowlist never
	// copies them, whatever the parent exports.
	allowlistPathVar = "PATH"
	allowlistHomeVar = "HOME"

	// contractEnvVarCount is the number of contract env vars ChildEnv sets
	// beyond the allowlist copies (ELECTRON_RUN_AS_NODE + the builtin and
	// personal provider config paths).
	contractEnvVarCount = 3

	// stderrCapDefault is the child stderr forward cap — the probe precedent
	// (app-server-probe.mjs STDERR_CAP_BYTES = 64 KiB, PITFALL 2). Beyond the
	// cap the pump keeps draining (silently) so the pipe buffer never fills.
	stderrCapDefault = 64 << 10

	// stderrChunkSize bounds one drain read.
	stderrChunkSize = 4096

	// waitDelayDefault bounds exec.Cmd.Wait's wait for pipe readers after the
	// process exits (stdlib WaitDelay semantics) — the backstop that stops a
	// stalled stdout reader from hanging the reap forever.
	waitDelayDefault = 5 * time.Second
)

// processTree is the platform-owned lifetime boundary for a backend and its
// descendants: Unix uses a process group; Windows uses a Job Object.
type processTree interface {
	assign(proc *os.Process) error
	resume() error
	terminate() error
	close()
}

// Child is one spawned app-server child process and the handles its
// supervisor needs: the protocol pipes, the process-tree owner, and the exit
// monitor's delivery.
type Child struct {
	// Cmd is the exec handle. Wait bookkeeping is owned by the monitor
	// inside SpawnChild — exactly one Wait per path (PITFALL 3).
	Cmd *exec.Cmd
	// Stdin is the protocol's write side; the supervisor holds it open and
	// closing it is the first stage of the shutdown sequence (the child
	// drains ~100 ms on EOF by design, dialect §1 :30).
	Stdin io.WriteCloser
	// Stdout is the protocol's read side — handed to the codec Conn, whose
	// read loop is the stdout pump (SpawnChild never reads Stdout itself).
	Stdout io.ReadCloser
	// PgID is the child's process-group id on Unix and its pid on Windows.
	// The platform process tree, not this diagnostic number, owns termination.
	PgID int
	// tree owns the platform process-group/Job-Object lifetime boundary.
	tree processTree
	// Exited delivers exactly one ExitInfo when the monitor has reaped the
	// child (buffered, so the monitor never blocks). Receive it only after
	// Done closes — the delivery is a single send, Done is the multi-observer
	// completion signal.
	Exited <-chan ExitInfo
	// Done closes when the monitor has reaped the child — observable by any
	// number of watchers (close is not consumed, unlike the Exited delivery).
	Done <-chan struct{}
	// StderrDone closes when the capped stderr pump has finished — the
	// Wait-ordering guard of PITFALL 3 (never reap before pumps finish).
	StderrDone <-chan struct{}
}

// ExitInfo is the monitor's one delivery: the reaped child's Wait error
// (nil on a clean exit; *exec.ExitError carries the exit status).
type ExitInfo struct {
	Err error
}

// SpawnOpts tunes one SpawnChild call. Zero values take production defaults.
type SpawnOpts struct {
	// WaitDelay bounds exec.Cmd.Wait's wait for pipe readers after the
	// process exits (0 takes the production default).
	WaitDelay time.Duration
	// StderrCap caps the child stderr bytes forwarded to the wrapper's
	// stderr (0 takes the 64 KiB probe cap of PITFALL 2).
	StderrCap int
	// StdoutTap, when set, receives the child's stdout instead of a
	// caller-read pipe: SpawnChild installs it as cmd.Stdout and leaves
	// Child.Stdout nil (helper-only — the mount owner never reads
	// Child.Stdout). exec's own copy goroutine drains the tap to EOF and
	// Wait joins that goroutine, so every line written immediately before
	// exit is in the tap when a failure error is built; WaitDelay is only
	// the close backstop, never the drain.
	StdoutTap io.Writer
	// StderrTap, when set, additionally receives the child's stderr while
	// the capped forward to the wrapper's stderr keeps its default behavior.
	// The tap owns its own cap (the mount owner passes a cappedBuffer), so a
	// chatty helper can never grow the wrapper's memory (T-10-05).
	StderrTap io.Writer
}

// SpawnArgv returns the canonical spawn argv for an Install — exactly
// [Binary, Entrypoint, "app-server", "--stdio"] per the canonical spawn line
// (install-discovery.md :48-52; the contract Phase 4's front leg inherits).
func SpawnArgv(inst Install) []string {
	return []string{inst.Binary, inst.Entrypoint, appServerSubcommand, stdioFlag}
}

// ChildEnv builds the child environment from parentEnv via the explicit
// allowlist (D-03 / PITFALL 9 / T-3-08): PATH and HOME copied from the
// parent, ELECTRON_RUN_AS_NODE=1 and ZCODE_BUILTIN_PROVIDER_CONFIG_FILE set
// from the Install, everything else — the proxy variables in particular —
// never crosses. The child's OBSERVED env is proven by TestSpawnEnv.
func ChildEnv(inst Install, parentEnv []string) []string {
	parent := make(map[string]string, len(parentEnv))

	for _, kv := range parentEnv {
		if k, v, ok := strings.Cut(kv, "="); ok {
			parent[k] = v
		}
	}

	env := make([]string, 0, len(allowlistKeys())+contractEnvVarCount)

	for _, key := range allowlistKeys() {
		if v, ok := envValue(parent, key); ok {
			env = append(env, key+"="+v)
		}
	}

	env = append(env, childRunAsNodeVar+"="+childRunAsNodeValue)

	if inst.BuiltinFile != "" {
		env = append(env, childBuiltinVar+"="+inst.BuiltinFile)
	}

	// The personal path pairs with the builtin one (resolveNodeProviderRuntimePaths
	// throws unless both are present): the desktop passes the same pair
	// (provider-runtime-env.ts:88-99). Derived from the child's own home —
	// the desktop's dataBaseDir default — never from the wrapper's cwd.
	home := homeValue(parent)
	if home != "" {
		env = append(env, childPersonalVar+"="+filepath.Join(home, ".zcode", "v2", "provider_config.json"))
	}

	return env
}

// SpawnChild spawns the child described by argv under env: its own process
// group (Setpgid), all three stdio pipes wired, the child's stderr drained by
// one goroutine forwarding capped to the wrapper's stderr (probe precedent,
// PITFALL 2), Stdout handed back as the codec Conn's read side, and an exit
// monitor that reaps exactly once after the pumps finish (PITFALL 3).
// Context cancellation and WaitDelay kill the whole process group, never
// just the direct child (PITFALL 6).
func SpawnChild(ctx context.Context, argv, env []string, opts SpawnOpts) (*Child, error) {
	if len(argv) == 0 || argv[0] == "" {
		return nil, errEmptyArgv
	}

	opts.applyDefaults()

	tree, err := newProcessTree()
	if err != nil {
		return nil, err
	}

	cmd, pipes, err := newSpawnCommand(ctx, argv, env, opts, tree)
	if err != nil {
		tree.close()

		return nil, err
	}

	if err := cmd.Start(); err != nil {
		pipes.closeAll()
		tree.close()

		return nil, fmt.Errorf("zcode: spawn %s: %w", argv[0], err)
	}
	if err := tree.assign(cmd.Process); err != nil {
		abortStartedChild(cmd, pipes, tree)

		return nil, err
	}
	if err := tree.resume(); err != nil {
		abortStartedChild(cmd, pipes, tree)

		return nil, err
	}

	exited := make(chan ExitInfo, 1)
	stderrDone := make(chan struct{})
	done := make(chan struct{})

	child := &Child{
		Cmd:        cmd,
		Stdin:      pipes.stdin,
		Stdout:     pipes.stdout,
		PgID:       cmd.Process.Pid,
		tree:       tree,
		Exited:     exited,
		Done:       done,
		StderrDone: stderrDone,
	}

	go drainChildStderr(pipes.stderr, opts.StderrCap, opts.StderrTap, stderrDone)
	go monitorChild(cmd, tree, stderrDone, exited, done)

	slog.Debug("supervisor: child spawned", "pid", child.PgID, "binary", argv[0])

	return child, nil
}

// abortStartedChild cleans up a process that Start created but the spawn path
// could not fully initialize. No monitor owns Wait on this path, so cleanup
// must reap exactly once before releasing the platform process-tree handle.
func abortStartedChild(cmd *exec.Cmd, pipes spawnPipes, tree processTree) {
	if termErr := tree.terminate(); termErr != nil {
		slog.Warn("supervisor: abort process-tree terminate", "pid", cmd.Process.Pid, "err", termErr)
	}
	_ = cmd.Process.Kill()
	pipes.closeAll()
	_ = cmd.Wait()
	tree.close()
}

// spawnPipes holds the three stdio pipe ends SpawnChild opens on the parent
// side, so a failed Start can close them all.
type spawnPipes struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// closeAll closes every parent-side pipe end (a failed Start would otherwise
// leak file descriptors). Stdout is nil when SpawnOpts.StdoutTap owns the
// stream.
func (p spawnPipes) closeAll() {
	_ = p.stdin.Close()
	if p.stdout != nil {
		_ = p.stdout.Close()
	}
	_ = p.stderr.Close()
}

// newSpawnCommand builds the exec.Cmd: env, the process-group start, the
// three stdio pipes, the group-kill Cancel, and the WaitDelay backstop.
func newSpawnCommand(
	ctx context.Context, argv, env []string, opts SpawnOpts, tree processTree,
) (*exec.Cmd, spawnPipes, error) {
	//nolint:gosec // argv is the supervisor's own spawn contract (the canonical
	// app-server line or the mount helper), never shell-interpreted user input
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env
	configureCommand(cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, spawnPipes{}, fmt.Errorf("zcode: spawn stdin pipe: %w", err)
	}

	var stdout io.ReadCloser

	if opts.StdoutTap != nil {
		// exec's copier drains the tap to EOF and Wait joins it: a caller-owned
		// StdoutPipe must never be relied on for that drain (WaitDelay closes it
		// and can discard buffered text).
		cmd.Stdout = opts.StdoutTap
	} else {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return nil, spawnPipes{}, fmt.Errorf("zcode: spawn stdout pipe: %w", err)
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, spawnPipes{}, fmt.Errorf("zcode: spawn stderr pipe: %w", err)
	}

	// Context cancellation and WaitDelay kill the WHOLE group (PITFALL 6) —
	// exec's default Cancel kills only the direct child, stranding Electron
	// helpers. cmd.Process is assigned inside Start before the ctx-watcher
	// goroutine exists, so this closure never races a nil Process.
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		if err := tree.assign(cmd.Process); err != nil {
			return err
		}

		return tree.terminate()
	}
	cmd.WaitDelay = opts.WaitDelay

	return cmd, spawnPipes{stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

// monitorChild reaps the child exactly once: it waits for the stderr pump to
// finish (PITFALL 3 — never Wait before the pumps finish), then Wait, then
// delivers ExitInfo on the buffered channel and closes Done (the multi-
// observer reap signal — close, unlike the delivery, is never consumed).
func monitorChild(
	cmd *exec.Cmd, tree processTree, stderrDone <-chan struct{}, exited chan<- ExitInfo, done chan struct{},
) {
	defer tree.close()
	<-stderrDone

	err := cmd.Wait()
	exited <- ExitInfo{Err: err}
	close(done)
}

// drainChildStderr forwards the child's stderr to the wrapper's stderr,
// capped (PITFALL 2): beyond the cap the pump keeps draining silently so the
// 64 KB pipe buffer never stalls the child mid-turn. A non-nil tap (the mount
// owner's cappedBuffer) mirrors the forwarded bytes for the caller's
// diagnostics; a nil tap preserves the app-server path byte-for-byte.
func drainChildStderr(rc io.Reader, capBytes int, tap io.Writer, done chan<- struct{}) {
	defer close(done)

	buf := make([]byte, stderrChunkSize)
	forwarded := 0

	for {
		n, readErr := rc.Read(buf)
		if n > 0 && forwarded < capBytes {
			room := capBytes - forwarded
			out := buf[:n]

			if n > room {
				out = buf[:room]
				slog.Warn("supervisor: child stderr exceeded the forward cap; draining silently", "cap", capBytes)
			}

			_, _ = os.Stderr.Write(out)
			forwarded += len(out)

			if tap != nil {
				_, _ = tap.Write(out)
			}
		}

		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				slog.Warn("supervisor: child stderr drain ended", "err", readErr)
			}

			return
		}
	}
}

// terminateTree is the staged shutdown's unconditional process-tree kill.
// The platform owner tolerates an already-gone tree.
func (c *Child) terminateTree() error {
	if c == nil || c.tree == nil {
		return nil
	}

	return c.tree.terminate()
}

// applyDefaults resolves SpawnOpts zero values to production defaults.
func (o *SpawnOpts) applyDefaults() {
	if o.StderrCap <= 0 {
		o.StderrCap = stderrCapDefault
	}

	if o.WaitDelay <= 0 {
		o.WaitDelay = waitDelayDefault
	}
}
