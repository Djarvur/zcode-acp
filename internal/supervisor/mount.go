// Mount owner (plan 10-01; MNT-01..03, MNT-06; D-01/D-02/D-11): a packed
// AppImage is mounted by a supervised `--appimage-mount` helper and the
// mounted tree is handed to the existing supervisor as an Install. No
// extraction: the AppImage runtime's own read-only FUSE mount is the only
// acquisition path, so no squashfs-root/zcode-<ver> copy is ever created
// (MNT-06).
//
// The helper is a SpawnChild child — own process group, capped stderr drain,
// exactly-one reap — but it is deliberately NOT owned by Supervisor.Start:
// the restart loop would remount on app-server crashes and could respawn
// against a vanished root (D-05). The lifecycle here is: create a private
// 0700 TMPDIR, spawn, read the mount line, validate the tree, watch for loss,
// and stage the teardown (SIGTERM, grace, group SIGKILL, reap, prove the
// mount is gone, remove only the private base).
//
// Protocol facts (RESEARCH Patterns 1, 2, 4, 6): the runtime prints the
// realpath of the mount directory to stdout after the mount is ready and then
// pauses forever; failure text also goes to stdout, so readiness scans for a
// validated path line instead of trusting the first line; TMPDIR relocates
// the mount; APPIMAGE_EXTRACT_AND_RUN in the environment silently turns the
// mount into extraction, which is why the helper env is an allowlist by
// construction. The runtime does not read stdin — stdin EOF is never a stop
// mechanism.

package supervisor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Mount lifecycle facts (RESEARCH Patterns 1, 2, 4, 6; D-02/D-04).
const (
	// mountBasePattern names the private 0700 TMPDIR we own. Every .mount_*
	// entry under it is ours (the runtime derives the mount dir from TMPDIR),
	// so a scan of the shared /tmp is never needed (T-10-03).
	mountBasePattern = "zcode-acp-mount-"
	// mountDirPrefix is the runtime's mount directory prefix
	// ($TMPDIR/.mount_<argv0-basename>XXXXXX); the basename varies, the
	// prefix is the only stable part.
	mountDirPrefix = ".mount_"
	// mountHelperFlag is the runtime's mount subcommand.
	mountHelperFlag = "--appimage-mount"
	// mountTmpdirVar is the helper's only output-routing variable.
	mountTmpdirVar = "TMPDIR"
	// mountPathFallback is the PATH the helper receives when the parent
	// carries none: the standard system binary dirs (a helper without a
	// usable PATH cannot resolve libfuse's fusermount).
	mountPathFallback = "/usr/local/bin:/usr/bin:/bin"
	// mountOutputCap bounds each captured helper stream (T-10-05: the 64 KiB
	// stderr precedent).
	mountOutputCap = 64 << 10
	// mountLineCap bounds the partial-line buffer when a broken helper writes
	// without a newline.
	mountLineCap = 4096
	// mountExecMask is the executable-bit mask the mount root's zcode must
	// carry.
	mountExecMask = 0o111

	// Production timings (RESEARCH A9); tests inject milliseconds.
	defaultMountReadyTimeout = 15 * time.Second
	defaultMountStopGrace    = 5 * time.Second
	defaultMountReapDeadline = 10 * time.Second
	defaultMountPollInterval = 50 * time.Millisecond
	defaultMountLossPoll     = 1 * time.Second
)

// Mount sentinels — wrapped with the dynamic context at the failure site,
// never constructed there (err113).
var (
	// ErrMountLost reports that the managed mount vanished while the wrapper
	// still owned the session (helper death or tree loss, D-05).
	ErrMountLost = errors.New("zcode: appimage mount lost")
	// errMountInstall reports an install that does not name a packed AppImage.
	errMountInstall = errors.New("zcode: mount: the install does not name a packed AppImage")
	// errMountBase reports an unusable private TMPDIR.
	errMountBase = errors.New("zcode: mount: private TMPDIR unusable")
	// errMountSpawn reports a helper that could not be spawned.
	errMountSpawn = errors.New("zcode: mount: helper spawn failed")
	// errMountReady reports a missing or rejected mount line.
	errMountReady = errors.New("zcode: mount: mount readiness failed")
	// errMountTree reports a mount that does not carry the spawnable tree.
	errMountTree = errors.New("zcode: mount: mounted tree unusable")
	// errMountTeardown reports an incomplete helper/mount teardown.
	errMountTeardown = errors.New("zcode: mount: teardown incomplete")
)

// MountOpts tunes one mount lifecycle. Zero values take production defaults;
// tests inject millisecond values so no test ever waits on the wall clock.
type MountOpts struct {
	// ReadyTimeout bounds the wait for the mount line after the spawn.
	ReadyTimeout time.Duration
	// StopGrace is the SIGTERM-to-group-SIGKILL grace on teardown.
	StopGrace time.Duration
	// ReapDeadline bounds the post-kill reap wait and the mount-gone poll.
	ReapDeadline time.Duration
	// PollInterval is the mount-gone poll interval.
	PollInterval time.Duration
	// LossPoll is the accessibility sample interval of the Lost watcher.
	LossPoll time.Duration
}

// DefaultMountOpts is the production MountOpts.
func DefaultMountOpts() MountOpts {
	return MountOpts{
		ReadyTimeout: defaultMountReadyTimeout,
		StopGrace:    defaultMountStopGrace,
		ReapDeadline: defaultMountReapDeadline,
		PollInterval: defaultMountPollInterval,
		LossPoll:     defaultMountLossPoll,
	}
}

// withDefaults resolves zero fields to the production defaults.
func (o MountOpts) withDefaults() MountOpts {
	if o.ReadyTimeout <= 0 {
		o.ReadyTimeout = defaultMountReadyTimeout
	}

	if o.StopGrace <= 0 {
		o.StopGrace = defaultMountStopGrace
	}

	if o.ReapDeadline <= 0 {
		o.ReapDeadline = defaultMountReapDeadline
	}

	if o.PollInterval <= 0 {
		o.PollInterval = defaultMountPollInterval
	}

	if o.LossPoll <= 0 {
		o.LossPoll = defaultMountLossPoll
	}

	return o
}

// Mount owns one packed-AppImage mount: the helper child, the private base,
// the validated mount root, and the staged teardown.
type Mount struct {
	appImage  string
	version   string
	root      string
	base      string
	builtin   string
	helperPID int
	opts      MountOpts
	child     *Child
	cancel    context.CancelFunc
	stderr    cappedBuffer
	lost      chan struct{}
	lostOnce  sync.Once
	stopOnce  sync.Once
	stopErr   error
}

// PrepareMount mounts the packed AppImage named by inst and validates the
// mounted tree (D-02). The private base is created 0700, the helper runs with
// the allowlisted env and the caller's cancellation detached (the app-server's
// shutdown must never SIGKILL the helper mid-readiness), and the mount line is
// accepted only when it is an absolute .mount_* directory one component under
// that base. On any failure the helper is stopped, the just-created base is
// removed, and the error names the AppImage, the failing step, and the capped
// observed output (D-03; the fuller failure classification is 10-02).
func PrepareMount(ctx context.Context, inst Install, opts MountOpts) (*Mount, error) {
	opts = opts.withDefaults()

	if inst.AppImage == "" {
		return nil, fmt.Errorf("%w: root %q", errMountInstall, inst.Root)
	}

	base, err := os.MkdirTemp("", mountBasePattern)
	if err != nil {
		return nil, fmt.Errorf("%w: create the private TMPDIR for %s: %w", errMountBase, inst.AppImage, err)
	}

	if problem := basePathProblem(base); problem != "" {
		_ = os.RemoveAll(base)

		return nil, fmt.Errorf("%w: %s produced %q, which %s (TMPDIR may not contain NUL, CR, or LF)",
			errMountBase, mountTmpdirVar, base, problem)
	}

	m := &Mount{
		appImage: inst.AppImage,
		version:  inst.Version,
		base:     base,
		builtin:  inst.BuiltinFile,
		opts:     opts,
		lost:     make(chan struct{}),
	}

	helperCtx, helperCancel := context.WithCancel(context.WithoutCancel(ctx))
	m.cancel = helperCancel

	tap := newMountTap(base)

	child, spawnErr := SpawnChild(helperCtx,
		[]string{inst.AppImage, mountHelperFlag},
		helperEnv(base),
		SpawnOpts{StdoutTap: tap, StderrTap: &m.stderr, WaitDelay: waitDelayDefault})
	if spawnErr != nil {
		helperCancel()
		_ = os.RemoveAll(base)

		return nil, fmt.Errorf("%w: spawn the %s helper for %s: %w",
			errMountSpawn, mountHelperFlag, inst.AppImage, spawnErr)
	}

	m.child = child
	m.helperPID = child.PgID

	root, readyErr := m.awaitMountLine(ctx, tap)
	if readyErr != nil {
		_ = m.Stop()

		return nil, fmt.Errorf("zcode: mount %s: %w (observed output %q)",
			inst.AppImage, readyErr, m.observedOutput(tap))
	}

	if treeErr := validateMountTree(root); treeErr != nil {
		_ = m.Stop()

		return nil, fmt.Errorf("zcode: mount %s: %w (observed output %q)",
			inst.AppImage, treeErr, m.observedOutput(tap))
	}

	m.root = root

	if m.builtin == "" {
		m.builtin = bundledBuiltinPath(root)
	}

	go m.watch(helperCtx)

	slog.Info("supervisor: appimage mounted",
		"appimage", m.appImage, "version", m.version, "mount", m.root, "pid", m.helperPID)

	return m, nil
}

// awaitMountLine waits for the helper to settle the readiness state (an
// accepted path, a rejected path, an exit, the timeout, or caller
// cancellation). The tap is fed by exec's stdout copier, which Wait joins, so
// the observed text is complete whenever the helper's Done fires.
func (m *Mount) awaitMountLine(ctx context.Context, tap *mountTap) (string, error) {
	timer := time.NewTimer(m.opts.ReadyTimeout)
	defer timer.Stop()

	select {
	case <-tap.settled:
		root, reject := tap.accepted()
		if reject != nil {
			return "", fmt.Errorf("%w: %w", errMountReady, reject)
		}

		return root, nil
	case <-m.child.Done:
		return "", fmt.Errorf("%w: the helper exited before the mount line (exit %s)",
			errMountReady, exitDescription(m.child))
	case <-timer.C:
		return "", fmt.Errorf("%w: no mount line within %v", errMountReady, m.opts.ReadyTimeout)
	case <-ctx.Done():
		return "", fmt.Errorf("%w: %w", errMountReady, ctx.Err())
	}
}

// exitDescription renders the reaped helper's exit for diagnostics.
func exitDescription(child *Child) string {
	select {
	case info := <-child.Exited:
		if info.Err == nil {
			return "0"
		}

		return info.Err.Error()
	default:
		return "unknown"
	}
}

// observedOutput joins the capped stdout and stderr captures for a failure
// error; the caller renders it with %q so a hostile helper cannot forge log
// lines (T-10-05).
func (m *Mount) observedOutput(tap *mountTap) string {
	return tap.observedText() + m.stderr.String()
}

// Install returns the mounted install: the mount root replaces the packed
// install's empty Root/Binary/Entrypoint, AppImage/Version are preserved, and
// BuiltinFile is the discovery-provided path or the bundled copy read from
// inside the mount.
func (m *Mount) Install() Install {
	return Install{
		Root:        m.root,
		Binary:      filepath.Join(m.root, binaryName),
		Entrypoint:  filepath.Join(m.root, filepath.FromSlash(entrypointRel)),
		BuiltinFile: m.builtin,
		Layout:      layoutAppImage,
		AppImage:    m.appImage,
		Version:     m.version,
	}
}

// AppImage returns the packed AppImage path the mount serves.
func (m *Mount) AppImage() string {
	return m.appImage
}

// Root returns the validated mount root.
func (m *Mount) Root() string {
	return m.root
}

// Lost closes once when the helper dies or the mount tree stops being
// accessible (D-05): the session must end with an explicit cause, never
// respawn against a vanished root.
func (m *Mount) Lost() <-chan struct{} {
	return m.lost
}

// Accessible reports whether the mount root still carries its zcode entry.
func (m *Mount) Accessible() bool {
	if m == nil || m.root == "" {
		return false
	}

	_, err := os.Stat(filepath.Join(m.root, binaryName))

	return err == nil
}

// Guard returns ErrMountLost when the mount is gone; a nil mount is healthy.
func (m *Mount) Guard() error {
	if m == nil {
		return nil
	}

	if m.Accessible() {
		return nil
	}

	return fmt.Errorf("%w: %s (mount %s)", ErrMountLost, m.appImage, m.root)
}

// HelperPID returns the helper's PID (also its process-group id). It is set
// on a successful PrepareMount and deliberately NOT cleared by Stop, so tests
// can assert ESRCH without ever signalling process group 0.
func (m *Mount) HelperPID() int {
	return m.helperPID
}

// Stop performs the idempotent staged helper teardown and returns the first
// teardown error: SIGTERM the helper's process group (never an already-reaped
// one — PID-reuse safety), wait StopGrace, escalate to a group SIGKILL, wait
// ReapDeadline, poll the mount root's zcode entry until it is gone, then
// remove only the private base. It never depends on fusermount and never
// closes the helper's stdin as a stop mechanism (the runtime pauses; it does
// not read stdin).
func (m *Mount) Stop() error {
	if m == nil {
		return nil
	}

	m.stopOnce.Do(func() { m.stopErr = m.stop() })

	return m.stopErr
}

// stop runs the staged teardown once (guarded by Stop's sync.Once).
func (m *Mount) stop() error {
	var errs []error

	if helperErr := m.stopHelper(); helperErr != nil {
		errs = append(errs, helperErr)
	}

	if m.root != "" {
		if goneErr := m.waitMountGone(); goneErr != nil {
			errs = append(errs, goneErr)
		}
	}

	if m.cancel != nil {
		m.cancel()
	}

	if removeErr := os.RemoveAll(m.base); removeErr != nil {
		errs = append(errs, fmt.Errorf("%w: remove the private base %s: %w", errMountTeardown, m.base, removeErr))
	}

	return errors.Join(errs...)
}

// stopHelper stages the helper half of the teardown. The helper does not read
// stdin, so the app-server's stdin-EOF stage does not apply: SIGTERM first,
// then the unconditional group SIGKILL backstop, then the bounded reap. An
// already-reaped helper is never signalled (its PID may be reused).
func (m *Mount) stopHelper() error {
	if m.child == nil {
		return nil
	}

	defer func() { _ = m.child.Stdin.Close() }()

	select {
	case <-m.child.Done:
		return nil
	default:
	}

	if termErr := syscall.Kill(-m.child.PgID, syscall.SIGTERM); termErr != nil &&
		!errors.Is(termErr, syscall.ESRCH) {
		slog.Warn("supervisor: mount helper SIGTERM", "pgid", m.child.PgID, "err", termErr)
	}

	select {
	case <-m.child.Done:
		return nil
	case <-time.After(m.opts.StopGrace):
	}

	// Re-check under the escalation: a helper that exited exactly at the grace
	// boundary must never be signalled again (PID reuse).
	select {
	case <-m.child.Done:
		return nil
	default:
		if killErr := killGroup(m.child.PgID); killErr != nil {
			slog.Warn("supervisor: mount helper group kill", "pgid", m.child.PgID, "err", killErr)
		}
	}

	select {
	case <-m.child.Done:
		return nil
	case <-time.After(m.opts.ReapDeadline):
		return fmt.Errorf("%w: the helper %d was not reaped within %v",
			errMountTeardown, m.helperPID, m.opts.ReapDeadline)
	}
}

// waitMountGone polls the mount root's zcode entry until it disappears
// (ENOENT/ENOTDIR) or the deadline passes. The mountpoint directory can
// outlive the unmount by tens of milliseconds, so the mount is verified, never
// the directory (RESEARCH Pitfall 4).
func (m *Mount) waitMountGone() error {
	probe := filepath.Join(m.root, binaryName)
	deadline := time.Now().Add(m.opts.ReapDeadline)

	for {
		_, err := os.Stat(probe)
		if err != nil && (errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("%w: the mount %s is still present; unmount it manually once the "+
				"helper is reaped (fusermount -u %s)", errMountTeardown, m.root, m.root)
		}

		time.Sleep(m.opts.PollInterval)
	}
}

// watch closes Lost once on helper death or tree loss.
func (m *Mount) watch(ctx context.Context) {
	ticker := time.NewTicker(m.opts.LossPoll)
	defer ticker.Stop()

	for {
		select {
		case <-m.lost:
			return
		case <-m.child.Done:
			m.markLost()

			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !m.Accessible() {
				m.markLost()

				return
			}
		}
	}
}

// markLost closes the Lost channel exactly once.
func (m *Mount) markLost() {
	m.lostOnce.Do(func() { close(m.lost) })
}

// basePathProblem reports why a private base path cannot be a helper TMPDIR
// ("" when it is usable): the helper's line is newline-delimited and
// NUL-terminated at the C boundary, so those bytes would corrupt or truncate
// it (T-10-03). A bad base is rejected before any spawn.
func basePathProblem(base string) string {
	switch {
	case strings.ContainsRune(base, 0):
		return "contains a NUL byte"
	case strings.ContainsRune(base, '\n'):
		return "contains a newline"
	case strings.ContainsRune(base, '\r'):
		return "contains a carriage return"
	default:
		return ""
	}
}

// helperEnv builds the mount helper's environment: an allowlist by
// construction (PATH, HOME, TMPDIR) so APPIMAGE_EXTRACT_AND_RUN,
// TARGET_APPIMAGE, the loader variables, and every credential/proxy variable
// can never turn the mount into extraction or code injection (T-10-01;
// RESEARCH Pattern 1, Pitfall 5). TMPDIR is the private base, which makes the
// mount point attributable to this wrapper run.
func helperEnv(base string) []string {
	env := make([]string, 0, 3)

	path := os.Getenv(allowlistPathVar)
	if path == "" {
		path = mountPathFallback
	}

	env = append(env, allowlistPathVar+"="+path)

	if home := os.Getenv(allowlistHomeVar); home != "" {
		env = append(env, allowlistHomeVar+"="+home)
	}

	env = append(env, mountTmpdirVar+"="+base)

	return env
}

// validateMountLine accepts line only when it is an absolute, existing,
// non-symlink directory named .mount_* one component under the private base;
// the whole line is the path (a TMPDIR with spaces is legal — never
// strings.Fields, RESEARCH Pitfall 6). A rejected path-looking line fails
// readiness immediately (T-10-02): the helper's stdout is untrusted.
func validateMountLine(base, resolvedBase, line string) (string, error) {
	clean := filepath.Clean(line)
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("%w: %q is not an absolute path", errMountReady, line)
	}

	if dir := filepath.Dir(clean); dir != base && dir != resolvedBase {
		return "", fmt.Errorf("%w: %q is not one .mount_* component under the private base %q",
			errMountReady, line, base)
	}

	if !strings.HasPrefix(filepath.Base(clean), mountDirPrefix) {
		return "", fmt.Errorf("%w: %q is not named %s*", errMountReady, line, mountDirPrefix)
	}

	info, err := os.Lstat(clean)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %w", errMountReady, line, err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%w: %q is a symlink", errMountReady, line)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%w: %q is not a directory", errMountReady, line)
	}

	return clean, nil
}

// validateMountTree proves the mount carries the layout-invariant spawn
// surface: an executable regular <mount>/zcode and a regular
// <mount>/resources/glm/zcode.cjs (install-discovery.md :38).
func validateMountTree(root string) error {
	binary := filepath.Join(root, binaryName)

	info, err := os.Stat(binary)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", errMountTree, binary, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", errMountTree, binary)
	}

	if info.Mode().Perm()&mountExecMask == 0 {
		return fmt.Errorf("%w: %s is not executable", errMountTree, binary)
	}

	entrypoint := filepath.Join(root, filepath.FromSlash(entrypointRel))

	entryInfo, err := os.Stat(entrypoint)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", errMountTree, entrypoint, err)
	}

	if !entryInfo.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", errMountTree, entrypoint)
	}

	return nil
}

// bundledBuiltinPath returns the bundled builtin copy inside the mount, or ""
// when the mount carries none (the honest empty result).
func bundledBuiltinPath(root string) string {
	path := filepath.Join(root, filepath.FromSlash(bundledProviderRel), builtinFileName)

	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
		return path
	}

	return ""
}

// cappedBuffer is a mutex-protected byte sink bounded to limit bytes: a broken
// or chatty helper can never stall the child or grow the wrapper's memory
// (T-10-05). Write always claims the whole chunk so the copier keeps
// draining; bytes beyond the limit are dropped.
type cappedBuffer struct {
	mu    sync.Mutex
	buf   []byte
	limit int
}

// Write implements io.Writer over the capped buffer.
func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.limit <= 0 {
		b.limit = mountOutputCap
	}

	if room := b.limit - len(b.buf); room > 0 {
		if len(p) > room {
			p = p[:room]
		}

		b.buf = append(b.buf, p...)
	}

	return n, nil
}

// String snapshots the captured bytes.
func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return string(b.buf)
}

// mountTap is the helper stdout tap: it splits the stream into whole lines
// (never field-splitting the path), records every non-empty line capped into
// the observed buffer, and settles readiness on the first accepted path or the
// first rejected absolute line. Write is called from exec's stdout copier and
// must never block.
type mountTap struct {
	mu       sync.Mutex
	base     string
	resolved string
	root     string
	reject   error
	observed cappedBuffer
	partial  []byte
	settled  chan struct{}
	once     sync.Once
}

// newMountTap builds the tap; the private base is resolved once (a mount line
// may carry the realpath of a symlinked TMPDIR).
func newMountTap(base string) *mountTap {
	resolved := base
	if target, err := filepath.EvalSymlinks(base); err == nil {
		resolved = target
	}

	return &mountTap{base: base, resolved: resolved, settled: make(chan struct{})}
}

// Write implements io.Writer over the helper's stdout.
func (t *mountTap) Write(p []byte) (int, error) {
	n := len(p)

	t.mu.Lock()
	defer t.mu.Unlock()

	t.partial = append(t.partial, p...)

	for {
		idx := bytes.IndexByte(t.partial, '\n')
		if idx < 0 {
			break
		}

		line := string(t.partial[:idx])
		t.partial = t.partial[idx+1:]
		t.handle(line)
	}

	// A helper that never terminates its line must not grow the buffer
	// unbounded; the tail is what a path line would need.
	if len(t.partial) > mountLineCap {
		t.partial = t.partial[len(t.partial)-mountLineCap:]
	}

	return n, nil
}

// handle classifies one line: it strips the trailing CR only (spaces inside a
// path are significant), records it as observed output, and settles on the
// first absolute path line — accepted when it validates, rejected otherwise.
func (t *mountTap) handle(line string) {
	line = strings.TrimSuffix(line, "\r")
	if line == "" {
		return
	}

	_, _ = t.observed.Write([]byte(line + "\n"))

	if t.root != "" || t.reject != nil {
		return
	}

	if !filepath.IsAbs(line) {
		// Failure text (the runtime prints it to stdout too) is diagnostics,
		// not a candidate path.
		return
	}

	root, err := validateMountLine(t.base, t.resolved, line)
	if err != nil {
		t.reject = err
		t.once.Do(func() { close(t.settled) })

		return
	}

	t.root = root
	t.once.Do(func() { close(t.settled) })
}

// accepted reports the settled readiness state.
func (t *mountTap) accepted() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.root, t.reject
}

// observedText snapshots the capped observed output.
func (t *mountTap) observedText() string {
	return t.observed.String()
}
