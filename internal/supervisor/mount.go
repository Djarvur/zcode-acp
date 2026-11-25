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
	"sync/atomic"
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
	// helperEnvVars is the size of the helper env allowlist (PATH, HOME,
	// TMPDIR).
	helperEnvVars = 3
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

	// fuseSuperblockMagic is the Linux FUSE superblock type
	// (FUSE_SUPER_MAGIC, 0x65735546). It is the fallback presence predicate
	// for a mount whose <root>/zcode entry is absent: a tree that failed
	// validation still has an active FUSE mount (D-03 teardown safety).
	fuseSuperblockMagic = 0x65735546

	// Remediation families (MNT-04): every failure error carries one. The
	// FUSE family names BOTH the setup checks and the package because newer
	// static type-2 runtimes do not need libfuse2 (T-10-08 / RESEARCH
	// Pitfall 16).
	remediationFUSE = "install FUSE support (/dev/fuse present and unprivileged " +
		"mounts allowed) and libfuse2: 'sudo apt install libfuse2t64' on Ubuntu 24.04 " +
		"or 'sudo apt install libfuse2' on older releases; newer static AppImage " +
		"runtimes do not need libfuse2"
	remediationImage = "verify the AppImage is a valid, readable, executable type-2 " +
		"AppImage ('chmod +x <appimage>') and that it is not truncated"
	remediationTmpdir  = "verify TMPDIR is writable and the helper can execute the AppImage"
	remediationGeneric = "verify the AppImage is executable, the helper can run it, and " +
		"that FUSE/libfuse2 is available"
)

// Mount failure steps, named verbatim in every classification error (D-03).
const (
	stepSpawnHelper   = "spawn the helper"
	stepReadMountLine = "read the mount line"
	stepValidateTree  = "validate the mount tree"
	stepTeardown      = "teardown"
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

// MountFailure reports whether err carries a structural mount failure rather
// than a bare caller cancellation: an unusable install/private base, a helper
// spawn failure, an unusable mounted tree, or an incomplete teardown. A
// readiness error that wraps only the caller's cancellation is NOT a mount
// failure — a signal during readiness may exit cleanly — but the same
// cancellation joined with a failed teardown is, because errors.Is still sees
// the teardown sentinel through the join. PrepareMount deliberately joins the
// teardown error into the acquisition error; callers must not discard it just
// because the signal context is cancelled (CR-01).
func MountFailure(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, errMountInstall) || errors.Is(err, errMountBase) ||
		errors.Is(err, errMountSpawn) || errors.Is(err, errMountTree) ||
		errors.Is(err, errMountTeardown) {
		return true
	}

	if !errors.Is(err, errMountReady) {
		return false
	}

	// A readiness failure that only reports the caller's cancellation is a
	// clean signal shutdown, not a mount failure.
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

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
	// Statfs reports path's filesystem type; nil takes syscall.Statfs. The
	// seam exists because a tree that failed validation has no zcode entry,
	// so an active FUSE mount is the fallback presence predicate — and only
	// a test can fake the FUSE superblock without FUSE (RESEARCH Pattern 4).
	Statfs func(path string) (fsType int64, err error)
}

// DefaultMountOpts is the production MountOpts.
func DefaultMountOpts() MountOpts {
	return MountOpts{
		ReadyTimeout: defaultMountReadyTimeout,
		StopGrace:    defaultMountStopGrace,
		ReapDeadline: defaultMountReapDeadline,
		PollInterval: defaultMountPollInterval,
		LossPoll:     defaultMountLossPoll,
		Statfs:       defaultStatfs,
	}
}

// defaultStatfs is the production filesystem-type predicate. The explicit
// conversion normalizes Darwin's uint32 and Linux's int64 Type fields at the
// MountOpts.Statfs int64 seam.
func defaultStatfs(path string) (int64, error) {
	var st syscall.Statfs_t

	if err := syscall.Statfs(path, &st); err != nil {
		return 0, fmt.Errorf("zcode: statfs %s: %w", path, err)
	}

	return int64(st.Type), nil
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

	if o.Statfs == nil {
		o.Statfs = defaultStatfs
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
	tap       *mountTap
	stderr    cappedBuffer
	lost      chan struct{}
	lostOnce  sync.Once
	stopping  atomic.Bool
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
		return nil, fmt.Errorf("%w: root %q; remediation: pass a packed .AppImage via --app-server",
			errMountInstall, inst.Root)
	}

	m, err := newMountBase(inst, opts)
	if err != nil {
		return nil, err
	}

	if startErr := m.start(ctx); startErr != nil {
		stopErr := m.Stop()
		if stopErr != nil {
			// The teardown detail rides alongside the readiness cause: a live
			// mount is never swallowed by the acquisition error.
			return nil, errors.Join(startErr, stopErr)
		}

		return nil, startErr
	}

	slog.Info("supervisor: appimage mounted",
		"appimage", m.appImage, "version", m.version, "mount", m.root, "pid", m.helperPID)

	return m, nil
}

// newMountBase creates the private 0700 base and the Mount shell; a base that
// contains NUL, CR, or LF is rejected and removed before any spawn (T-10-03).
func newMountBase(inst Install, opts MountOpts) (*Mount, error) {
	base, err := os.MkdirTemp("", mountBasePattern)
	if err != nil {
		return nil, fmt.Errorf("%w: create the private TMPDIR for %s: %w; remediation: %s",
			errMountBase, inst.AppImage, err, remediationTmpdir)
	}

	if problem := basePathProblem(base); problem != "" {
		_ = os.RemoveAll(base)

		return nil, fmt.Errorf("%w: %s produced %q, which %s (TMPDIR may not contain NUL, CR, or LF); "+
			"remediation: %s", errMountBase, mountTmpdirVar, base, problem, remediationTmpdir)
	}

	return &Mount{
		appImage: inst.AppImage,
		version:  inst.Version,
		base:     base,
		builtin:  inst.BuiltinFile,
		opts:     opts,
		lost:     make(chan struct{}),
	}, nil
}

// awaitMountLine waits for the helper to settle the readiness state (an
// accepted path, a rejected path, an exit, the timeout, or caller
// cancellation). The tap is fed by exec's stdout copier, which Wait joins, so
// the observed text is complete whenever the helper's Done fires.
func awaitMountLine(ctx context.Context, m *Mount) (string, error) {
	timer := time.NewTimer(m.opts.ReadyTimeout)
	defer timer.Stop()

	select {
	case <-m.tap.settled:
		root, reject := m.tap.accepted()
		if reject != nil {
			return "", m.failure(stepReadMountLine, reject)
		}

		return root, nil
	case <-m.child.Done:
		return "", m.failure(stepReadMountLine,
			fmt.Errorf("%w: the helper exited before the mount line (%s)",
				errMountReady, exitDescription(m.child)))
	case <-timer.C:
		return "", m.failure(stepReadMountLine,
			fmt.Errorf("%w: no mount line within %v", errMountReady, m.opts.ReadyTimeout))
	case <-ctx.Done():
		return "", m.failure(stepReadMountLine, fmt.Errorf("%w: %w", errMountReady, ctx.Err()))
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
// lines (T-10-05). The joined text is capped again so the two per-stream caps
// cannot sum into an unbounded diagnostic.
func observedOutput(m *Mount) string {
	out := m.tap.observedText() + m.stderr.String()
	if len(out) > mountOutputCap {
		out = out[:mountOutputCap]
	}

	return out
}

// classifyMountFailure maps the helper's observed output to the remediation
// family. The legacy runtime prints its failure text to STDOUT, so the
// strings are matched against the captured stdout+stderr (RESEARCH Pattern 2,
// Pitfall 16). The known strings are classified; the raw observed text always
// rides the error regardless, and a newer static runtime's unknown text
// degrades to the generic remediation rather than a wrong libfuse2 diagnosis.
func classifyMountFailure(observed string) string {
	switch {
	case containsAny(observed,
		"dlopen(): error loading libfuse.so.2",
		"Cannot mount AppImage, please check your FUSE setup.",
		"AppImages require FUSE to run."):
		return "FUSE/libfuse2 is unavailable: " + remediationFUSE
	case containsAny(observed,
		"Failed to get fs offset",
		"This doesn't look like a squashfs image",
		"Failed to open AppImage file"):
		return "the AppImage is not a readable type-2 squashfs image: " + remediationImage
	case containsAny(observed,
		"create mount dir error",
		"open dir error",
		"permission denied"):
		return "the mount directory or file permissions are unusable: " + remediationTmpdir
	default:
		return remediationGeneric
	}
}

// containsAny reports whether s contains any of the needles.
func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(s, needle) {
			return true
		}
	}

	return false
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

// failure builds the four-part mount error (D-03/MNT-04): the AppImage path,
// the failing step, the capped observed output rendered with %q, and a
// classification-derived remediation. Every readiness, validation, spawn, and
// teardown error on the mount path is built here.
func (m *Mount) failure(step string, cause error) error {
	observed := observedOutput(m)

	return fmt.Errorf("zcode: mount %s: %s: %w (observed output %q); remediation: %s",
		m.appImage, step, cause, observed, classifyMountFailure(observed))
}

// start spawns the helper under a cancellation-detached context, reads the
// mount line, validates the mounted tree, fills the bundled builtin, and arms
// the Lost watcher. The caller owns the failure teardown (Stop).
func (m *Mount) start(ctx context.Context) error {
	helperCtx, helperCancel := context.WithCancel(context.WithoutCancel(ctx))
	m.cancel = helperCancel
	m.tap = newMountTap(m.base)

	child, spawnErr := SpawnChild(helperCtx,
		[]string{m.appImage, mountHelperFlag},
		helperEnv(m.base),
		SpawnOpts{StdoutTap: m.tap, StderrTap: &m.stderr, WaitDelay: waitDelayDefault})
	if spawnErr != nil {
		helperCancel()
		_ = os.RemoveAll(m.base)

		return m.failure(stepSpawnHelper, fmt.Errorf("%w: %w", errMountSpawn, spawnErr))
	}

	m.child = child
	m.helperPID = child.PgID

	root, readyErr := awaitMountLine(ctx, m)
	if readyErr != nil {
		return readyErr
	}

	if treeErr := validateMountTree(root); treeErr != nil {
		return m.failure(stepValidateTree, treeErr)
	}

	m.root = root

	if m.builtin == "" {
		m.builtin = bundledBuiltinPath(root)
	}

	go m.watch(helperCtx)

	return nil
}

// stop runs the staged teardown once (guarded by Stop's sync.Once). The
// stopping flag is set BEFORE the helper is signalled: an intentional
// teardown must never close Lost and masquerade as a mid-session loss (D-05).
func (m *Mount) stop() error {
	m.stopping.Store(true)

	var errs []error

	if helperErr := m.stopHelper(); helperErr != nil {
		errs = append(errs, helperErr)
	}

	goneErr := m.waitMountGone()
	if goneErr != nil {
		errs = append(errs, goneErr)
	}

	if m.cancel != nil {
		m.cancel()
	}

	// RemoveAll is only safe once every mount is proven gone: traversing a
	// live FUSE mount would walk the AppImage's read-only tree (or block). A
	// failed proof retains the 0700 base for a manual unmount (D-03).
	if goneErr == nil {
		if removeErr := os.RemoveAll(m.base); removeErr != nil {
			errs = append(errs, fmt.Errorf("%w: remove the private base %s: %w",
				errMountTeardown, m.base, removeErr))
		}
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

// waitMountGone proves the mount is gone (D-02's zero-stale-mounts rule). A
// validated mount is probed through <root>/zcode with the FUSE superblock as
// the fallback for a tree that failed validation; a mount whose line never
// arrived is probed by scanning the private base's .mount_* children for an
// active FUSE mount. The mountpoint directory can outlive the unmount by tens
// of milliseconds, so the mount is verified, never the directory (RESEARCH
// Pitfall 4).
func (m *Mount) waitMountGone() error {
	if m.root == "" {
		return m.waitBaseMountsGone()
	}

	deadline := time.Now().Add(m.opts.ReapDeadline)

	for {
		if !m.mountPresent(m.root) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("%w: the mount %s is still present; remediation: %s",
				errMountTeardown, m.root, m.unmountRemediation(m.root))
		}

		time.Sleep(m.opts.PollInterval)
	}
}

// waitBaseMountsGone polls the private base for active FUSE mounts whose line
// never arrived. A plain directory (the fake runtime, or a tree that unmounted
// cleanly) is gone; only the FUSE superblock keeps the base retained.
func (m *Mount) waitBaseMountsGone() error {
	deadline := time.Now().Add(m.opts.ReapDeadline)

	for {
		live, err := m.liveBaseMounts()
		if err != nil {
			return fmt.Errorf("%w: scan the private base %s: %w", errMountTeardown, m.base, err)
		}

		if len(live) == 0 {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("%w: the mount %s is still present; remediation: %s",
				errMountTeardown, strings.Join(live, ", "), m.unmountRemediation(live[0]))
		}

		time.Sleep(m.opts.PollInterval)
	}
}

// liveBaseMounts lists the private base's .mount_* children that are still
// active FUSE mounts.
func (m *Mount) liveBaseMounts() ([]string, error) {
	entries, err := os.ReadDir(m.base)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err //nolint:wrapcheck // wrapped by waitBaseMountsGone with the base path
	}

	live := make([]string, 0, len(entries))

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), mountDirPrefix) {
			continue
		}

		path := filepath.Join(m.base, entry.Name())
		if m.fuseMounted(path) {
			live = append(live, path)
		}
	}

	return live, nil
}

// mountPresent reports whether root is still a usable or active mount: the
// zcode entry present, a stale/unusable mount error, or the FUSE superblock
// (a tree that failed validation has no zcode but is still mounted).
func (m *Mount) mountPresent(root string) bool {
	_, err := os.Stat(filepath.Join(root, binaryName))

	switch {
	case err == nil:
		return true
	case errors.Is(err, syscall.ENOTCONN), errors.Is(err, syscall.ENOTDIR):
		return true
	case !errors.Is(err, os.ErrNotExist):
		return true
	}

	return m.fuseMounted(root)
}

// fuseMounted reports whether path carries the Linux FUSE superblock type.
func (m *Mount) fuseMounted(path string) bool {
	fsType, err := m.opts.Statfs(path)
	if err != nil {
		return false
	}

	return fsType == fuseSuperblockMagic
}

// unmountRemediation names the manual recovery once the helper is reaped: the
// runtime holds the mount's fd while it lives, so no external unmount can
// succeed before that (RESEARCH Pattern 4, Pitfall 18).
func (m *Mount) unmountRemediation(root string) string {
	return fmt.Sprintf("the helper is reaped; unmount the mount manually (fusermount -u %s) "+
		"and remove the retained private base %s", root, m.base)
}

// watch closes Lost once on helper death or tree loss. The stopping flag is
// checked before every loss verdict: Stop's own SIGTERM kills the helper and
// may remove the tree, and that intentional teardown is not a loss.
func (m *Mount) watch(ctx context.Context) {
	ticker := time.NewTicker(m.opts.LossPoll)
	defer ticker.Stop()

	for {
		select {
		case <-m.lost:
			return
		case <-m.child.Done:
			if !m.stopping.Load() {
				m.markLost()
			}

			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if m.stopping.Load() {
				return
			}

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
	env := make([]string, 0, helperEnvVars)

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
