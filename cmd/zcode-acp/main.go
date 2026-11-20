// Command zcode-acp exposes ZCode Desktop as a standard ACP (Agent Client
// Protocol) v1 server over stdio. The wiring (plan 04-05): supervisor
// discovery resolves the app-server install (the --app-server override seam
// wins first — the test/manual seam and relocated-install override), the
// Phase-3 supervisor owns the child lifecycle (health watch, crash window,
// staged shutdown), a zcode.Conn rides the child's exec pipes, and the
// acpfront Front (wrapped via the acpport seam) serves ACP v1 over THIS
// process's own stdio — stdout carries protocol frames ONLY, every log line
// goes to the stderr slog (ARCHITECTURE Anti-Pattern 2).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// version is reported by --version and advertises as the agentInfo version
// (D-02 real agentInfo — the initialize report carries the binary's own
// version) until the release flow stamps a real value (Phase 7).
var version = "0.0.0-dev"

// Environment fallbacks for the flags (the flag always wins over its env var,
// so a wrapper config env and a CLI flag compose like a prefix/override pair).
const (
	envAppServer      = "ZCODE_ACP_APP_SERVER"
	envModel          = "ZCODE_ACP_MODEL"
	envReasoningLevel = "ZCODE_ACP_REASONING_LEVEL"

	// optDir is the package-family discovery root (the deb layout, /opt/ZCode).
	optDir = "/opt/ZCode"

	// spawnWait bounds the wait for the supervisor's first spawned child.
	spawnWait = 10 * time.Second

	// shutdownWatchdog bounds the staged-shutdown wait (the supervisor's own
	// stages are bounded; this guards OUR wait on its completion).
	shutdownWatchdog = 15 * time.Second

	// exitInvalidConfig is the usage exit code: the flags/env combination is
	// malformed (distinct from the runtime-failure exit 1 — the fake
	// binary's ExitUsage precedent: invalid input exits non-zero, naming
	// the contract).
	exitInvalidConfig = 2
)

// Configuration and lifecycle sentinels — wrapped with the dynamic context
// at the failure site, never constructed there (err113).
var (
	errLevelWithoutModel = errors.New("--reasoning-level set without --model")
	errModelShape        = errors.New("--model must be providerId/modelId")
	errSpawnTimeout      = errors.New("app-server spawn timed out")
	errShutdownWatchdog  = errors.New("staged shutdown did not complete within the watchdog")
	errChildLost         = errors.New("app-server exited before the ACP client disconnected")
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	appServerFlag := flag.String("app-server", "",
		"path to a ZCode install root (or app-server binary) overriding discovery — also "+envAppServer)
	modelFlag := flag.String("model", "", "backend model selection as providerId/modelId — also "+envModel)
	reasoningFlag := flag.String("reasoning-level", "", "reasoning level for --model — also "+envReasoningLevel)
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintln(os.Stdout, "zcode-acp "+version)

		return
	}

	// stdout is the ACP protocol from here on — the default slog handler is
	// re-pointed at stderr so every log line (ours, the SDK's, the
	// supervisor's) stays off the protocol stream.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	cfg, err := parseConfig(*appServerFlag, *modelFlag, *reasoningFlag)
	if err != nil {
		slog.Error("zcode-acp: invalid configuration", "err", err)

		os.Exit(exitInvalidConfig)
	}

	if err := run(cfg); err != nil {
		slog.Error("zcode-acp: exiting after failure", "err", err)

		os.Exit(1)
	}
}

// config is the resolved launch configuration: the app-server override path
// ("" = production discovery) and the optional model selection (nil = the
// session/new leg skips the setModel follow-up — setModel iff configured).
type config struct {
	appServer string
	model     *acpfront.ModelSelection
}

// parseConfig resolves flags over their env fallbacks (flags win) and
// validates the combination: a model selection must parse providerId/modelId,
// and a reasoning level without a model is a configuration error.
func parseConfig(appServerFlag, modelFlag, reasoningFlag string) (config, error) {
	cfg := config{appServer: firstNonEmpty(appServerFlag, os.Getenv(envAppServer))}

	level := firstNonEmpty(reasoningFlag, os.Getenv(envReasoningLevel))
	selection := firstNonEmpty(modelFlag, os.Getenv(envModel))

	if selection == "" {
		if level != "" {
			return config{}, fmt.Errorf("%w (%s)", errLevelWithoutModel, envModel)
		}

		return cfg, nil
	}

	provider, model, ok := strings.Cut(selection, "/")
	if !ok || provider == "" || model == "" {
		return config{}, fmt.Errorf("%w, got %q", errModelShape, selection)
	}

	cfg.model = &acpfront.ModelSelection{
		ProviderID:     provider,
		ModelID:        model,
		ReasoningLevel: level,
	}

	return cfg, nil
}

// firstNonEmpty returns the first non-empty string, "" when all are empty.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

// run composes the minimal chat loop over the production lifecycle: discover,
// supervise, connect the back leg, serve ACP over os.Stdin/os.Stdout, and
// stage-shut-down when the client disconnects or the child exits.
func run(cfg config) error {
	ctx := context.Background()

	inst, err := discoverInstall(ctx, cfg.appServer)
	if err != nil {
		return err
	}

	// Two lifecycles, deliberately: the supervision ctx drives the
	// supervisor's Start (and its staged teardown), while the spawn ctx is
	// the exec-level kill backstop cancelled only AFTER the staged teardown
	// completed. Cancelling both together would race: SpawnChild wires
	// cmd.Cancel to the group SIGKILL on its ctx, so a shared cancellation
	// would SIGKILL the child the instant the teardown began — the graceful
	// stdin-EOF drain (dialect §1, the child's ~100 ms clean exit) would
	// never run, and the child would always die mid-flight.
	supCtx, supCancel := context.WithCancel(ctx)
	spawnCtx, spawnCancel := context.WithCancel(ctx)
	defer spawnCancel()

	child, startDone, err := startSupervised(supCtx, spawnCtx, supCancel, inst)
	if err != nil {
		return err
	}

	// The Conn and the Front (whose delta tap registers on the conn's
	// ordered fan-out BEFORE any machine exists) are constructed before any
	// frame flows — the Conn hook contract.
	conn := zcode.NewConn(childDuplex{reader: child.Stdout, writer: child.Stdin})
	defer func() { _ = conn.Close() }()

	opts := []acpfront.Option{acpfront.WithVersion(version)}
	if cfg.model != nil {
		opts = append(opts, acpfront.WithModel(*cfg.model))
	}

	front := acpfront.New(conn, opts...)

	// The ACP agent connection owns os.Stdin/os.Stdout from here: the SDK
	// reads requests and writes every response frame — nothing else touches
	// stdout.
	agentConn, _, _ := acpport.NewAgentSideConnection(acpport.NewAgent(front), os.Stdout, os.Stdin)

	slog.Info("zcode-acp: serving ACP v1 over stdio", "version", version)

	childLost := false

	select {
	case <-agentConn.Done():
		slog.Info("zcode-acp: ACP client disconnected")
	case <-child.Done:
		childLost = true
	}

	// Staged shutdown always: the supervision cancel drives the supervisor's
	// staged teardown (stdin end → grace → unconditional group SIGKILL →
	// bounded reap — stdin-EOF is never trusted, PITFALL 6); the conn's
	// deferred Close joins the read loop after.
	supCancel()

	select {
	case err := <-startDone:
		if err != nil {
			return fmt.Errorf("supervisor shutdown: %w", err)
		}
	case <-time.After(shutdownWatchdog):
		return errShutdownWatchdog
	}

	spawnCancel() // the teardown backstop: the group is long gone; ESRCH-quiet

	if childLost {
		return errChildLost
	}

	slog.Info("zcode-acp: shutdown complete")

	return nil
}

// discoverInstall resolves the app-server install: the --app-server override
// (documented order step 1) when set, production discovery otherwise. A
// discovery failure returns the Phase-3 actionable diagnostics — one probe
// row per attempted probe, never a guess (install-discovery.md :34).
func discoverInstall(ctx context.Context, appServer string) (supervisor.Install, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	probers := supervisor.DefaultProbers(home, optDir)
	if appServer != "" {
		probers.Override = func() (supervisor.Install, bool) {
			return overrideInstall(appServer), true
		}
	}

	inst, probes, err := probers.Discover(ctx)
	if err != nil {
		// Wrapped: the discovery diagnostics ride intact (%w keeps
		// errors.Is(ErrNoInstall)) — the probe rows are the message body.
		return supervisor.Install{}, fmt.Errorf("discover app-server install: %w", err)
	}

	for _, row := range probes {
		slog.Debug("zcode-acp: discovery probe", "probe", row.Name, "result", row.Result)
	}

	slog.Info("zcode-acp: app-server resolved", "binary", inst.Binary, "layout", inst.Layout)

	return inst, nil
}

// overrideInstall maps the --app-server path into the discovery Install
// shape: a directory is an install root validated against the invariant
// spawn surface (ExplicitRootInstall — <root>/zcode +
// <root>/resources/glm/zcode.cjs, install-discovery.md :38); a regular file
// is the app-server binary itself, its entrypoint still the invariant path
// beside it. The shape is handed through as named — a wrong shape fails loud
// at spawn with exec's actionable error, never a silent guess.
func overrideInstall(path string) supervisor.Install {
	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		if inst, ok := supervisor.ExplicitRootInstall(path); ok {
			return inst
		}

		return supervisor.Install{
			Root:   path,
			Binary: filepath.Join(path, "zcode"),
			Layout: "explicit",
		}
	}

	root := filepath.Dir(path)

	return supervisor.Install{
		Root:       root,
		Binary:     path,
		Entrypoint: filepath.Join(root, "resources", "glm", "zcode.cjs"),
		Layout:     "explicit",
	}
}

// startSupervised starts the supervisor over the SpawnFunc seam and waits
// for the first spawned child; the returned channel carries Start's outcome
// (nil on the intentional shutdown path). The spawn seam hands SpawnChild
// the Phase-3 canonical argv (SpawnArgv) and allowlisted env (ChildEnv —
// the proxy vars scrubbed by construction), under the SEPARATE spawn ctx
// whose cancellation is the post-teardown kill backstop (see run).
func startSupervised(
	supCtx context.Context, spawnCtx context.Context, supCancel context.CancelFunc, inst supervisor.Install,
) (*supervisor.Child, <-chan error, error) {
	spawnCh := make(chan *supervisor.Child, 1)
	startDone := make(chan error, 1)

	sup := supervisor.NewSupervisor(func(_ context.Context) (*supervisor.Child, error) {
		child, spawnErr := supervisor.SpawnChild(spawnCtx,
			supervisor.SpawnArgv(inst),
			supervisor.ChildEnv(inst, os.Environ()),
			supervisor.SpawnOpts{})
		if spawnErr == nil {
			spawnCh <- child
		}

		// Wrapped: Start owns the supervisor-context framing of this error
		// ("zcode: supervisor: spawn app-server"); the exec detail rides %w.
		if spawnErr != nil {
			return nil, fmt.Errorf("spawn the app-server child: %w", spawnErr)
		}

		return child, nil
	}, supervisor.DefaultPolicy())

	go func() { startDone <- sup.Start(supCtx) }()

	select {
	case child := <-spawnCh:
		return child, startDone, nil
	case err := <-startDone:
		supCancel()

		return nil, startDone, fmt.Errorf("spawn app-server: %w", err)
	case <-time.After(spawnWait):
		supCancel()

		return nil, startDone, errSpawnTimeout
	}
}

// childDuplex joins the child's stdout read side and stdin write side into
// the one io.ReadWriteCloser the codec Conn consumes (the supervisor owns
// the pipes; the Conn rides them — the 03-09 e2eDuplex shape). Close
// unblocks a blocked Read, which is what Conn.Close relies on.
type childDuplex struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

// Read implements io.Reader over the child's stdout.
func (d childDuplex) Read(p []byte) (int, error) {
	//nolint:wrapcheck // the transport passthrough IS the contract: the codec
	// Conn owns error interpretation (io.EOF demux, single-writer ordering)
	// and a wrapper layer would corrupt every per-chunk error identity.
	return d.reader.Read(p)
}

// Write implements io.Writer to the child's stdin.
func (d childDuplex) Write(p []byte) (int, error) {
	//nolint:wrapcheck // the transport passthrough IS the contract (see Read).
	return d.writer.Write(p)
}

// Close ends both pipe halves, joining the conn's read loop.
func (d childDuplex) Close() error {
	return errors.Join(d.writer.Close(), d.reader.Close())
}
