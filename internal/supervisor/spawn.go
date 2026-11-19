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
	"strings"
	"syscall"
	"time"
)

// Spawn-contract facts (docs/protocol/install-discovery.md :48-52).
const (
	// childRunAsNodeVar/Value make the Electron binary run as plain Node.
	childRunAsNodeVar   = "ELECTRON_RUN_AS_NODE"
	childRunAsNodeValue = "1"
	// childBuiltinVar carries the newest zcode-builtin.json into the child.
	childBuiltinVar = "ZCODE_BUILTIN_PROVIDER_CONFIG_FILE"
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
	// beyond the allowlist copies (ELECTRON_RUN_AS_NODE + the builtin var).
	contractEnvVarCount = 2

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

// Child is one spawned app-server child process and the handles its
// supervisor needs: the protocol pipes, the process-group id, and the exit
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
	// PgID is the child's process-group id. Setpgid makes the child its own
	// group leader, so -PgID addresses the whole tree incl. grandchildren.
	PgID int
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
		if v, ok := parent[key]; ok {
			env = append(env, key+"="+v)
		}
	}

	env = append(env, childRunAsNodeVar+"="+childRunAsNodeValue)

	if inst.BuiltinFile != "" {
		env = append(env, childBuiltinVar+"="+inst.BuiltinFile)
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
	if len(argv) == 0 {
		return nil, errEmptyArgv
	}

	opts.applyDefaults()

	cmd, pipes, err := newSpawnCommand(ctx, argv, env, opts)
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		pipes.closeAll()

		return nil, fmt.Errorf("zcode: spawn %s: %w", argv[0], err)
	}

	exited := make(chan ExitInfo, 1)
	stderrDone := make(chan struct{})
	done := make(chan struct{})

	child := &Child{
		Cmd:        cmd,
		Stdin:      pipes.stdin,
		Stdout:     pipes.stdout,
		PgID:       cmd.Process.Pid,
		Exited:     exited,
		Done:       done,
		StderrDone: stderrDone,
	}

	go drainChildStderr(pipes.stderr, opts.StderrCap, stderrDone)
	go monitorChild(cmd, stderrDone, exited, done)

	slog.Debug("supervisor: child spawned", "pid", child.PgID, "binary", argv[0])

	return child, nil
}

// spawnPipes holds the three stdio pipe ends SpawnChild opens on the parent
// side, so a failed Start can close them all.
type spawnPipes struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// closeAll closes every parent-side pipe end (a failed Start would otherwise
// leak file descriptors).
func (p spawnPipes) closeAll() {
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	_ = p.stderr.Close()
}

// newSpawnCommand builds the exec.Cmd: env, the process-group start, the
// three stdio pipes, the group-kill Cancel, and the WaitDelay backstop.
func newSpawnCommand(ctx context.Context, argv, env []string, opts SpawnOpts) (*exec.Cmd, spawnPipes, error) {
	//nolint:gosec // argv is the supervisor's own SpawnArgv contract, never user input
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env
	// The process-group start: the child leads its own group so the staged
	// shutdown can SIGKILL the whole tree (T-3-09, PITFALL 6).
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, spawnPipes{}, fmt.Errorf("zcode: spawn stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, spawnPipes{}, fmt.Errorf("zcode: spawn stdout pipe: %w", err)
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

		return killGroup(cmd.Process.Pid)
	}
	cmd.WaitDelay = opts.WaitDelay

	return cmd, spawnPipes{stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

// monitorChild reaps the child exactly once: it waits for the stderr pump to
// finish (PITFALL 3 — never Wait before the pumps finish), then Wait, then
// delivers ExitInfo on the buffered channel and closes Done (the multi-
// observer reap signal — close, unlike the delivery, is never consumed).
func monitorChild(cmd *exec.Cmd, stderrDone <-chan struct{}, exited chan<- ExitInfo, done chan struct{}) {
	<-stderrDone

	err := cmd.Wait()
	exited <- ExitInfo{Err: err}
	close(done)
}

// drainChildStderr forwards the child's stderr to the wrapper's stderr,
// capped (PITFALL 2): beyond the cap the pump keeps draining silently so the
// 64 KB pipe buffer never stalls the child mid-turn.
func drainChildStderr(rc io.Reader, capBytes int, done chan<- struct{}) {
	defer close(done)

	buf := make([]byte, stderrChunkSize)
	forwarded := 0

	for {
		n, readErr := rc.Read(buf)
		if n > 0 && forwarded < capBytes {
			room := capBytes - forwarded
			out := buf

			if n > room {
				out = buf[:room]
				slog.Warn("supervisor: child stderr exceeded the forward cap; draining silently", "cap", capBytes)
			}

			_, _ = os.Stderr.Write(out)
			forwarded += len(out)
		}

		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				slog.Warn("supervisor: child stderr drain ended", "err", readErr)
			}

			return
		}
	}
}

// killGroup SIGKILLs the whole process group; an already-gone group
// (ESRCH) is success.
func killGroup(pgid int) error {
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("zcode: group kill %d: %w", pgid, err)
	}

	return nil
}

// allowlistKeys is the parent env allowlist (copy-only vars).
func allowlistKeys() []string {
	return []string{allowlistPathVar, allowlistHomeVar}
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
