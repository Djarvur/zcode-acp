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
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// version is reported by --version and advertises as the agentInfo version
// (D-02 real agentInfo — the initialize report carries the binary's own
// version). The value is the omp card's compatibility-floor stamp (plan
// 07-04, D-03): the client's version probe parses the answer as semver and
// blocks the whole card below 17.0.5, and model selectors resolve only from
// 18.1.9 — the stamp carries build metadata, honest about being this
// wrapper's build at the floor. The -ldflags override seam stays open for a
// release flow to stamp later.
var version = "18.1.9+zacp.0"

// Environment fallbacks for the flags (the flag always wins over its env var,
// so a wrapper config env and a CLI flag compose like a prefix/override pair).
const (
	envAppServer      = "ZCODE_ACP_APP_SERVER"
	envModel          = "ZCODE_ACP_MODEL"
	envReasoningLevel = "ZCODE_ACP_REASONING_LEVEL"

	//nolint:gosec // an env-var NAME (the OQ4 disposition names it in the
	// honest error), never a credential value.
	envAPIKey = "ZCODE_ACP_API_KEY"

	// optDir is the package-family discovery root (the deb layout, /opt/ZCode).
	optDir = "/opt/ZCode"

	// secretToolBin is the libsecret CLI the D-06 credential loader shells
	// (a system binary on the guarded real tier — never a Go module, the
	// T-5-SC zero-new-dependency posture).
	secretToolBin = "secret-tool"

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
	if ompRequested(os.Args[1:]) {
		runOmp(os.Args[1:])

		return
	}

	showVersion := flag.Bool("version", false, "print version and exit")
	appServerFlag := flag.String("app-server", "",
		"path to a ZCode install root (or app-server binary) overriding discovery — also "+envAppServer)
	modelFlag := flag.String("model", "", "backend model selection as providerId/modelId — also "+envModel)
	reasoningFlag := flag.String("reasoning-level", "", "reasoning level for --model — also "+envReasoningLevel)
	apiKeyFlag := flag.String("api-key", "",
		"z.ai API key for authenticate{zai-api-key} — also "+envAPIKey)
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintln(os.Stdout, "zcode-acp "+version)

		return
	}

	// stdout is the ACP protocol from here on — the default slog handler is
	// re-pointed at stderr so every log line (ours, the SDK's, the
	// supervisor's) stays off the protocol stream.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	// WR-03 (07-REVIEW): the flag form carries the key on the process argv —
	// world-readable via ps / /proc/<pid>/cmdline for the wrapper's whole
	// lifetime, strictly worse than the owner-only-readable environment the
	// env fallback uses. One stderr line (never stdout — the protocol
	// stream); the flag keeps working for compatibility.
	if *apiKeyFlag != "" {
		slog.Warn("zcode-acp: --api-key rides the process argv " +
			"(world-readable via ps and /proc/<pid>/cmdline while the wrapper runs) — " +
			"prefer the " + envAPIKey + " environment variable")
	}

	cfg, err := parseConfig(*appServerFlag, *modelFlag, *reasoningFlag, *apiKeyFlag)
	if err != nil {
		slog.Error("zcode-acp: invalid configuration", "err", err)

		os.Exit(exitInvalidConfig)
	}

	if err := run(cfg); err != nil {
		slog.Error("zcode-acp: exiting after failure", "err", err)

		os.Exit(1)
	}
}

// ompSubcommand is the omp card persona's dispatch token: the client
// launches the card as `<program> omp ...` (the per-agent command override
// is "<wrapper> omp" — the dialect doc's Launch contract), so the first
// non-flag positional equal to omp selects the omp arm. First-positional
// dispatch is the only switch (plan 07-04: root acp-mode behavior stays
// byte-identical for non-omp argv).
const ompSubcommand = "omp"

// ompRootValueConsumes reports whether the named root flag consumes a
// separate value token; the dispatch pre-scan skips those values so an
// argv like `--model omp` stays a root invocation instead of hijacking
// the arm.
func ompRootValueConsumes(name string) bool {
	switch name {
	case "app-server", "model", "reasoning-level", "api-key":
		return true
	}

	return false
}

// ompRequested reports whether the argv selects the omp card arm: the first
// non-flag positional equals omp (stdlib flag stops at the first positional
// — Pitfall 3 — so the dispatch must be recognized BEFORE flag.Parse). Root
// value flags consume their values during the scan and "--" terminates flag
// scanning (everything after it is positional).
func ompRequested(args []string) bool {
	flagsDone := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !flagsDone {
			if arg == "--" {
				flagsDone = true

				continue
			}

			if strings.HasPrefix(arg, "-") {
				name := strings.TrimLeft(arg, "-")
				if !strings.Contains(name, "=") && ompRootValueConsumes(name) && i+1 < len(args) {
					i++ // the flag's value token — never a positional
				}

				continue
			}
		}

		return arg == ompSubcommand
	}

	return false
}

// ompOptions is the parsed omp card argv: the version answer, the config
// flags the root arm shares (validated identically by parseConfig), and the
// acceptance counters for the card flags this wrapper takes and ignores
// (--plugin-dir, --session-dir) plus the unknown flags the client may add
// across versions (the omp contract's tolerance posture).
type ompOptions struct {
	version        bool
	appServer      string
	model          string
	reasoningLevel string
	apiKey         string

	pluginDirSet  bool
	sessionDirSet bool
	unknownCount  int
}

// ompFlagValues holds the omp FlagSet's parsed value pointers so ompParse's
// permissive loop can read each chunk's answers after its Parse.
type ompFlagValues struct {
	version        *bool
	appServer      *string
	model          *string
	reasoningLevel *string
	apiKey         *string
	pluginDir      *string
	sessionDir     *string
}

// ompFlagSet is the omp arm's flag set over the card argv surface: the
// config flags the root arm carries plus the card's own flags accepted and
// ignored. Output is discarded — a parse failure surfaces through ompParse's
// honest-error path, never a usage dump on stdout (stdout carries the probe
// answer and protocol frames only).
func ompFlagSet() (*flag.FlagSet, *ompFlagValues) {
	fs := flag.NewFlagSet(ompSubcommand, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	return fs, &ompFlagValues{
		version: fs.Bool("version", false, "print version and exit"),
		appServer: fs.String("app-server", "",
			"path to a ZCode install root (or app-server binary) overriding discovery"),
		model:          fs.String("model", "", "backend model selection as providerId/modelId"),
		reasoningLevel: fs.String("reasoning-level", "", "reasoning level for --model"),
		apiKey:         fs.String("api-key", "", "z.ai API key for authenticate{zai-api-key}"),
		pluginDir:      fs.String("plugin-dir", "", "accepted and ignored (the omp card visuals contract)"),
		sessionDir: fs.String("session-dir", "",
			"accepted and ignored (the client's disposable discovery launches)"),
	}
}

// ompParse re-tokenizes the post-omp args (Pitfall 3: the card's own argv
// carries its flags AFTER the `acp` positional — `omp acp --model X` —
// which stdlib flag can never parse in one pass). The loop parses flag
// chunks, skips bare positionals (the card's `acp` marker is never
// meaningful to this wrapper), and tolerates unknown flags: an unknown flag
// plus one following non-flag token (its value, when it carries one) is
// skipped and counted. A known-flag parse error (bad syntax, a missing
// value) is an honest error, never silence.
func ompParse(args []string) (ompOptions, error) {
	opts := ompOptions{}

	rest := args

	for len(rest) > 0 {
		fs, v := ompFlagSet()

		parseErr := fs.Parse(rest)

		if *v.version {
			opts.version = true
		}

		if *v.appServer != "" {
			opts.appServer = *v.appServer
		}

		if *v.model != "" {
			opts.model = *v.model
		}

		if *v.reasoningLevel != "" {
			opts.reasoningLevel = *v.reasoningLevel
		}

		if *v.apiKey != "" {
			opts.apiKey = *v.apiKey
		}

		opts.pluginDirSet = opts.pluginDirSet || *v.pluginDir != ""
		opts.sessionDirSet = opts.sessionDirSet || *v.sessionDir != ""

		rest = fs.Args()

		if parseErr != nil {
			if !isUnknownFlagError(parseErr) {
				return ompOptions{}, fmt.Errorf("omp: %w", parseErr)
			}

			// Tolerated: also skip the unknown flag's separate value token
			// when one follows (a non-dash token after an unknown flag).
			if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
				rest = rest[1:]
			}

			opts.unknownCount++

			continue
		}

		if len(rest) > 0 { // the first positional — the card's `acp` marker; ignored
			rest = rest[1:]
		}
	}

	return opts, nil
}

// isUnknownFlagError reports whether a FlagSet error is the undefined-flag
// class the omp contract tolerates (anything else — bad syntax, a missing
// value — is an honest parse error).
func isUnknownFlagError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "flag provided but not defined")
}

// runOmp is the omp card arm (plan 07-04, D-03): answer the client's
// version probe in any position under omp — the probe shape is
// `<program> <prefixArgs> --version`, prefix args included, card args
// absent — then bind the card's --model selection and serve the standard
// ACP mode. The re-tokenized argv feeds the SAME parseConfig validation
// (identical sentinels, env fallbacks, exitInvalidConfig usage class as
// the root arm); the accepted-and-ignored card flags and the tolerated
// unknowns surface as debug lines naming their contracts, and the launch
// binding logs ids and shapes only — never credential or env values
// (T-07-12).
func runOmp(args []string) {
	// stderr slog first: every omp-arm diagnostic (parse, config, runtime)
	// stays off stdout — stdout carries the probe answer / protocol only.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	opts, err := ompParse(args)
	if err != nil {
		slog.Error("zcode-acp omp: invalid arguments", "err", err)

		os.Exit(exitInvalidConfig)
	}

	if opts.version {
		_, _ = fmt.Fprintln(os.Stdout, "zcode-acp "+version)

		return
	}

	// Accepted-and-ignored card flags: one debug line each naming the
	// contract — the disposable discovery launch (--session-dir) and the
	// visuals-configured chat launch (--plugin-dir) must never break
	// startup.
	if opts.pluginDirSet {
		slog.Debug("zcode-acp omp: --plugin-dir accepted and ignored (the card visuals contract)")
	}

	if opts.sessionDirSet {
		slog.Debug("zcode-acp omp: --session-dir accepted and ignored (the client's disposable discovery launch)")
	}

	if opts.unknownCount > 0 {
		slog.Debug("zcode-acp omp: unknown card flags tolerated and ignored", "count", opts.unknownCount)
	}

	cfg, err := parseConfig(opts.appServer, opts.model, opts.reasoningLevel, opts.apiKey)
	if err != nil {
		slog.Error("zcode-acp omp: invalid configuration", "err", err)

		os.Exit(exitInvalidConfig)
	}

	if cfg.model != nil {
		// Launch binding notice: provider/model ids and whether the level
		// rode explicitly (the snapshot default fills it inside the front)
		// — shapes only, never credential or env values.
		slog.Info("zcode-acp omp: model selection bound at launch",
			"provider", cfg.model.ProviderID,
			"model", cfg.model.ModelID,
			"levelExplicit", cfg.model.ReasoningLevel != "")
	}

	if err := run(cfg); err != nil {
		slog.Error("zcode-acp: exiting after failure", "err", err)

		os.Exit(1)
	}
}

// config is the resolved launch configuration: the app-server override path
// ("" = production discovery), the optional model selection (nil = the
// session/new leg skips the setModel follow-up — setModel iff configured),
// and the resolved api key ("" = the authenticate{zai-api-key} arm answers
// the honest coded error naming ZCODE_ACP_API_KEY — never a fallback read
// of the desktop's config store, OQ4).
type config struct {
	appServer string
	model     *acpfront.ModelSelection
	apiKey    string
}

// parseConfig resolves flags over their env fallbacks (flags win) and
// validates the combination: a model selection must parse providerId/modelId,
// and a reasoning level without a model is a configuration error. A
// configured model WITHOUT a reasoning level keeps parsing — the IN-04
// omit-shape handles the wire (the setModel follow-up omits the options
// object so the backend's own actionable refusal surfaces for
// level-strict providers).
func parseConfig(appServerFlag, modelFlag, reasoningFlag, apiKeyFlag string) (config, error) {
	cfg := config{
		appServer: firstNonEmpty(appServerFlag, os.Getenv(envAppServer)),
		apiKey:    firstNonEmpty(apiKeyFlag, os.Getenv(envAPIKey)),
	}

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

// defaultReadbackModels is the authenticate read-back want list when no
// model is configured: the z.ai personal-layer catalog pair the corpus rows
// lock (zai-api/GLM-5.3-Flash, zai-api/GLM-5.3). A catalog drift fails
// authenticate HONESTLY — the read-back negative names the actionable check
// (verify basedOnZCodeBuiltinRevision against the active builtin) — never a
// fabricated success. A fresh slice per call keeps the contract immutable.
func defaultReadbackModels() []string {
	return []string{"zai-api/GLM-5.3-Flash", "zai-api/GLM-5.3"}
}

// readbackModels derives the authenticate read-back want list: the
// configured model selection when present (the personal layer must carry
// the model the wrapper binds), the documented catalog pair otherwise.
func readbackModels(model *acpfront.ModelSelection) []string {
	if model == nil {
		return defaultReadbackModels()
	}

	return []string{model.ProviderID + "/" + model.ModelID}
}

// credentialsPath is the desktop credentials file the login-keyring key is
// bound to (the field-verified secret-tool lookup's username derivation
// input: sha256 over the resolved path string). Missing home resolution
// yields an empty path — the loader fails loud at command build and the
// cipher ask degrades honestly (D-06), never a phantom-path lookup.
func credentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".zcode", "v2", "credentials.json")
}

// settingPath is the desktop setting.json the production entitlement gate
// reads (the ~/.zcode/v2/setting.json resolver joining credentialsPath's
// posture). Missing home resolution yields an empty path — the gate's
// missing-setting arm degrades honestly (familysel.go's T-06-16 posture),
// never a phantom read.
func settingPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".zcode", "v2", "setting.json")
}

// patDesktopVersionFallback is the reference's documented desktop-version
// fallback (ref installedAppVersion's "0.0.0") — patClientVersion's answer
// when the install tree carries no version-looking segment (A6: no
// server-side validation is documented; the real run confirms).
const patDesktopVersionFallback = "0.0.0"

// desktopVersionPattern digs the version-looking segment out of an install
// path segment (the layout names carry it — e.g. zcode-3.18.2/unpacked,
// ZCode-3.18.2.AppImage; the reference's installedAppVersion(appBundlePath())
// best-effort port). Best-effort by design: a false-positive segment would
// only shape the mint chain's clientVersion diagnostic field (A6 — the
// server validates nothing documented).
var desktopVersionPattern = regexp.MustCompile(`\d+\.\d+(\.\d+)?`)

// desktopVersion resolves the desktop version the PAT mint reports: the
// discovery-parsed Install.Version first (a mounted root carries no version
// segment), then the packed AppImage basename, then the layout path segments
// deepest-first — "" when nothing matches (the documented fallback applies).
func desktopVersion(inst supervisor.Install) string {
	if inst.Version != "" {
		return inst.Version
	}

	if v := versionFromPath(inst.AppImage); v != "" {
		return v
	}

	for _, path := range []string{inst.Binary, inst.Root, inst.Entrypoint} {
		if v := versionFromPath(path); v != "" {
			return v
		}
	}

	return ""
}

// versionFromPath digs a version-looking segment out of one install path,
// deepest segment first.
func versionFromPath(path string) string {
	segs := strings.Split(path, string(filepath.Separator))

	for i := len(segs) - 1; i >= 0; i-- {
		if v := desktopVersionPattern.FindString(segs[i]); v != "" {
			return v
		}
	}

	return ""
}

// patClientVersion resolves the mint chain's clientVersion seam input (the
// DESKTOP's version per the reference's posture): the best-effort install
// dig when present, the documented fallback otherwise. Lazy by contract —
// the minter calls it only when a mint actually runs.
func patClientVersion(inst supervisor.Install) func() string {
	return func() string {
		if v := desktopVersion(inst); v != "" {
			return v
		}

		return patDesktopVersionFallback
	}
}

// codingPlanMachinery is the composed coding-plan machinery (plan 06-07,
// D-12: NO new user configuration — the desktop login IS the credential;
// ZCODE_ACP_API_KEY stays the api-key credential unchanged). Every member
// executes lazily: the prober and the store's v2 loader exec only when
// authenticate/asks arrive, the minter's chain runs only when a mint is
// needed, the gate reads setting.json only when the overlay builds (D-13:
// CI never constructs a keyring call).
type codingPlanMachinery struct {
	creds  *interact.CredentialStore
	prober interact.KeyringProber
	gate   *interact.EntitlementGate
	minter *hostsvc.PatMinter
}

// buildCodingPlanMachinery constructs the coding-plan machinery over the
// production seams: the 06-01 credential store (the v2 decrypt loader over
// the SAME secret-tool binary + credentials path the cipher seam uses), the
// D-02 proactive keyring prober, the production entitlement gate over
// settingPath, and the 06-02 PAT minter (official origin default; the
// desktop-version clientVersion seam). Construction performs no I/O.
func buildCodingPlanMachinery(inst supervisor.Install) *codingPlanMachinery {
	env := interact.ScrubbedEnv(os.Environ())

	creds := interact.NewCredentialStore(credentialsPath(),
		interact.NewSecretToolKeyLoader(secretToolBin, credentialsPath(), env))

	return &codingPlanMachinery{
		creds:  creds,
		prober: interact.NewKeyringProber(secretToolBin, credentialsPath(), env),
		gate:   interact.NewEntitlementGate(settingPath(), creds),
		minter: hostsvc.NewPatMinter(
			hostsvc.WithPatClientVersion(patClientVersion(inst)),
			hostsvc.WithPatCredentials(creds),
		),
	}
}

// buildFront constructs the Front over the resolved config and install:
// the agentInfo version, the optional model selection, and the AUTH-01
// authenticate machinery (the key binding, the install for the snapshot
// formula, the derived read-back want list, and — plan 06-07 — the
// coding-plan seams: store, prober, gate, minter). The delta tap registers
// on the conn's ordered fan-out here — before any machine exists.
func buildFront(
	conn *zcode.Conn, cfg config, inst supervisor.Install, keys hostsvc.KeySource,
	machinery *codingPlanMachinery,
) *acpfront.Front {
	opts := []acpfront.Option{acpfront.WithVersion(version)}

	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		opts = append(opts,
			acpfront.WithCatalogRoots(acpfront.DefaultCatalogRoots(home)...),
			acpfront.WithCommandRoots(acpfront.DefaultCommandRoots(home)...),
		)
	}
	if cfg.model != nil {
		opts = append(opts, acpfront.WithModel(*cfg.model))
	}

	opts = append(opts, acpfront.WithAuth(acpfront.AuthConfig{
		Keys:       keys,
		Install:    inst,
		WantModels: readbackModels(cfg.model),
		Creds:      machinery.creds,
		Pat:        machinery.minter,
		Probe:      machinery.prober,
		Entitle:    machinery.gate.Entitled,
	}))

	return acpfront.New(conn, opts...)
}

// wireResponder builds the D-01 responder over the full policy table (plan
// 05-07 wiring; plan 06-07 swaps the runtime-headers option to the
// mode-aware PAT source) and assigns conn.OnRequest before the ACP
// connection exists and before any traffic starts — the Conn hook contract
// (frames flow only after the client speaks first; the app-server itself
// sends nothing unprompted at boot). The policy table carries the whole
// phase stack: the 05-02 permission bridge forwards
// interaction/requestPermission to the ACP client, the mode-aware source
// selects the recorded auth mode (D-11: api-key delegates to the static-key
// handler byte-compatibly, coding-plan serves the PAT answer from the
// in-process cache with the D-05 exactly-one re-mint and the scope
// refusal), and the D-06 decrypt seam answers real cipher asks from the
// login keyring — every injected seam EXECUTES only when its ask actually
// arrives (CI never runs this binary against a keyring; keyring-locked
// machines degrade to the honest coded declines).
func wireResponder(
	conn *zcode.Conn, front *acpfront.Front, keys hostsvc.KeySource, machinery *codingPlanMachinery,
	userInputAsk acpport.UserInputAsker,
) {
	responder := interact.NewResponder(
		interact.WithPermissionForwarder(interact.NewPermissionBridge(front.PermissionForward()).Forward),
		interact.WithUserInputAsker(userInputAsk),
		interact.WithRuntimeHeadersKeySource(hostsvc.PatHeadersSource(hostsvc.PatHeadersSourceDeps{
			Minter:         machinery.minter,
			Mode:           front.AuthMode,
			APIKeyFallback: hostsvc.RuntimeHeadersHandler(keys),
		})),
		interact.WithCredentialDecrypt(interact.NewCredentialDecrypt(
			interact.NewSecretToolKeyLoader(secretToolBin, credentialsPath(), interact.ScrubbedEnv(os.Environ())))),
	)

	conn.OnRequest = responder.Serve
}

// runOutcome is the first terminal observation of a ride; exit cause
// precedence is decided from it in finishRun.
type runOutcome struct {
	clientGone bool
	childLost  bool
	mountLost  bool
	signalled  bool
}

// run composes the minimal chat loop over the production lifecycle: discover,
// mount a packed AppImage when the install requires it, supervise, connect the
// back leg, serve ACP over os.Stdin/os.Stdout, and stage-shut-down when the
// client disconnects, the child exits, the mount is lost, or SIGINT/SIGTERM/
// SIGHUP arrives.
func run(cfg config) error {
	// signalCtx drives discovery, mount acquisition, and the supervisor's
	// shutdown. The spawn context below is deliberately NOT derived from it:
	// a signal must never SIGKILL the app-server before its staged stdin-EOF
	// drain (MNT-03, T-10-09).
	signalCtx, stopSignals := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stopSignals()

	inst, err := discoverInstall(signalCtx, cfg.appServer)
	if err != nil {
		if signalCtx.Err() != nil {
			slog.Info("zcode-acp: signal received during discovery; shutting down")

			return nil
		}

		return err
	}

	// Mount-first acquisition (MNT-01/MNT-02): a packed AppImage is mounted
	// before anything spawns. The deferred stop is the early-return backstop;
	// the normal path stops the mount explicitly after the supervisor's staged
	// shutdown (order is load-bearing: app-server first, helper second).
	inst, mount, err := acquireMount(signalCtx, inst)
	if err != nil {
		if signalCtx.Err() != nil {
			slog.Info("zcode-acp: signal received during mount acquisition; shutting down")

			return nil
		}

		return err
	}

	defer stopMountBackstop(mount)

	// Two lifecycles, deliberately: the supervision ctx drives the supervisor's
	// staged teardown, while the spawn ctx is the exec-level kill backstop
	// cancelled only AFTER that teardown completed (a shared cancellation would
	// SIGKILL the child before its graceful stdin-EOF drain). spawnCtx derives
	// from a non-signal context for the same reason.
	supCtx, supCancel := context.WithCancel(signalCtx)
	spawnCtx, spawnCancel := context.WithCancel(context.WithoutCancel(signalCtx))
	defer spawnCancel()

	child, startDone, err := startSupervised(supCtx, spawnCtx, supCancel, inst, mount)
	if err != nil {
		return err
	}

	outcome := serveACP(child, cfg, inst, mount, signalCtx.Done())

	return finishRun(outcome, startDone, supCancel, spawnCancel, mount)
}

// finishRun drives the staged shutdown and the exit-cause precedence. Order:
// the supervision cancel ends the supervisor's teardown (stdin end -> grace ->
// unconditional group SIGKILL -> bounded reap; stdin-EOF is never trusted,
// PITFALL 6); the mount is stopped only after the app-server is reaped and its
// group is gone (D-02's ordered teardown); the spawn cancel is the post-
// teardown kill backstop. A teardown error ALWAYS wins (exit 1) — a signal or
// client disconnect never masks a stale mount; only when the mount was proven
// gone does a signal/client disconnect count as a clean exit 0.
func finishRun(
	outcome runOutcome, startDone <-chan error, supCancel, spawnCancel context.CancelFunc, mount *supervisor.Mount,
) error {
	lostErr := mountLostCause(outcome.mountLost, mount)

	supCancel()

	startErr := awaitSupervisorShutdown(startDone)
	spawnCancel() // the teardown backstop: the group is long gone; ESRCH-quiet

	stopErr := stopMount(mount)

	return classifyRunExit(outcome, startErr, stopErr, lostErr)
}

// awaitSupervisorShutdown waits for the supervisor's staged teardown to
// return, bounded by the watchdog.
func awaitSupervisorShutdown(startDone <-chan error) error {
	select {
	case err := <-startDone:
		return err
	case <-time.After(shutdownWatchdog):
		return errShutdownWatchdog
	}
}

// mountLostCause captures the mount cause before Stop tears the mount down.
func mountLostCause(mountLost bool, mount *supervisor.Mount) error {
	if !mountLost || mount == nil {
		return nil
	}

	if guardErr := mount.Guard(); guardErr != nil {
		return fmt.Errorf("appimage mount is not usable: %w", guardErr)
	}

	return fmt.Errorf("%w: %s (mount %s)", supervisor.ErrMountLost, mount.AppImage(), mount.Root())
}

// classifyRunExit applies the exit-cause precedence: (1) a teardown error
// always wins; (2) a supervisor failure (other than the mount guard) fails
// loud; (3) only a proven-gone mount lets a signal or client disconnect exit
// cleanly; (4) a lost mount reports the mount cause; (5) a healthy-mount child
// loss reports errChildLost.
func classifyRunExit(outcome runOutcome, startErr, stopErr, lostErr error) error {
	if stopErr != nil {
		if startErr != nil && !errors.Is(startErr, supervisor.ErrMountLost) {
			return errors.Join(stopErr, fmt.Errorf("supervisor shutdown: %w", startErr))
		}

		return stopErr
	}

	if startErr != nil {
		if !errors.Is(startErr, supervisor.ErrMountLost) {
			return fmt.Errorf("supervisor shutdown: %w", startErr)
		}

		if lostErr == nil {
			lostErr = fmt.Errorf("supervisor shutdown: %w", startErr)
		}
	}

	if outcome.signalled || outcome.clientGone {
		slog.Info("zcode-acp: shutdown complete")

		return nil
	}

	if lostErr != nil {
		return lostErr
	}

	if outcome.childLost {
		return errChildLost
	}

	slog.Info("zcode-acp: shutdown complete")

	return nil
}

// stopMountBackstop is the deferred early-return backstop: Stop is idempotent,
// so after the normal-path stop it is a no-op, and it is also invoked on every
// return that happens before the shutdown tail.
func stopMountBackstop(mount *supervisor.Mount) {
	if err := stopMount(mount); err != nil {
		slog.Warn("zcode-acp: mount teardown backstop", "err", err)
	}
}

// serveACP wires the back-leg Conn, the Front (whose delta tap registers on the
// conn's ordered fan-out BEFORE any machine exists — the Conn hook contract),
// the D-01 responder, and the ACP agent connection over the child's pipes, then
// waits for the client to disconnect or the child to exit. It reports whether
// the app-server was lost before the client disconnected.
//
// One key, one source (AUTH-01): the resolved config binding feeds the
// authenticate machinery, the runtime-headers answer, and the snapshot builder
// alike — in-memory only, never logged, never committed (D-05); an empty key
// answers authenticate's honest coded error naming ZCODE_ACP_API_KEY (OQ4). The
// coding-plan machinery composes beside it (plan 06-07, D-12: the desktop login
// IS that path's credential — no new user configuration), lazily executing only
// when authenticate or asks arrive (D-13).
func serveACP(
	child *supervisor.Child, cfg config, inst supervisor.Install,
	mount *supervisor.Mount, signals <-chan struct{},
) runOutcome {
	conn := zcode.NewConn(childDuplex{reader: child.Stdout, writer: child.Stdin})
	defer func() { _ = conn.Close() }()

	keys := hostsvc.NewStaticKey(cfg.apiKey, "")
	machinery := buildCodingPlanMachinery(inst)
	front := buildFront(conn, cfg, inst, keys, machinery)

	// The ACP agent connection owns os.Stdin/os.Stdout from here: the SDK reads
	// requests and writes every response frame — nothing else touches stdout.
	agentConn, _, _ := acpport.NewAgentSideConnection(acpport.NewAgent(front), os.Stdout, os.Stdin)

	wireResponder(conn, front, keys, machinery, acpport.NewUserInputAsker(agentConn.CreateElicitation))

	slog.Info("zcode-acp: serving ACP v1 over stdio", "version", version)

	outcome := runOutcome{}

	var mountLost <-chan struct{}
	if mount != nil {
		mountLost = mount.Lost()
	}

	select {
	case <-agentConn.Done():
		slog.Info("zcode-acp: ACP client disconnected")
		outcome.clientGone = true
	case <-child.Done:
		outcome.childLost = true
	case <-mountLost:
		outcome.mountLost = true

		slog.Warn("zcode-acp: appimage mount lost",
			"appimage", mount.AppImage(), "mount", mount.Root())
	case <-signals:
		outcome.signalled = true

		slog.Info("zcode-acp: signal received; shutting down")
	}

	// A child loss with an inaccessible mount reports the mount cause: the
	// vanished root explains the child exit (RESEARCH Pattern 5).
	if outcome.childLost && mount != nil && !mount.Accessible() {
		outcome.mountLost = true
	}

	return outcome
}

// acquireMount performs the mount-first acquisition: a packed AppImage install
// (AppImage set, Root empty) is mounted by a supervised --appimage-mount
// helper and the mounted Install replaces it; every other install is handed
// through unchanged (a nil Mount means there is nothing to stop). The mount is
// stopped on every exit path — here on failure, by the caller's deferred
// backstop, and explicitly after the app-server's staged shutdown.
func acquireMount(ctx context.Context, inst supervisor.Install) (supervisor.Install, *supervisor.Mount, error) {
	if !inst.RequiresMount() {
		return inst, nil, nil
	}

	mount, err := supervisor.PrepareMount(ctx, inst, supervisor.DefaultMountOpts())
	if err != nil {
		return supervisor.Install{}, nil, fmt.Errorf("mount the app-server AppImage: %w", err)
	}

	slog.Info("zcode-acp: appimage backend mounted",
		"appimage", mount.AppImage(), "version", inst.Version, "mount", mount.Root(), "pid", mount.HelperPID())

	installed := mount.Install()

	return installed, mount, nil
}

// stopMount stops a mount-backed acquisition (nil-safe) and reports a
// teardown failure as a runtime failure: a mount left behind is never a
// warning-only outcome (A5).
func stopMount(mount *supervisor.Mount) error {
	if mount == nil {
		return nil
	}

	if err := mount.Stop(); err != nil {
		return fmt.Errorf("stop the app-server AppImage mount: %w", err)
	}

	slog.Info("zcode-acp: appimage mount stopped", "appimage", mount.AppImage(), "mount", mount.Root())

	return nil
}

// discoverInstall resolves the app-server install: the --app-server override
// (documented order step 1) when set, production discovery otherwise. The
// override accepts an install root, an app-server binary, or a packed
// .AppImage (D-04) — the mapping lives in supervisor.OverrideInstall so the
// manual real tier shares one implementation. A discovery failure returns the
// Phase-3 actionable diagnostics — one probe row per attempted probe, never a
// guess (install-discovery.md :34).
func discoverInstall(ctx context.Context, appServer string) (supervisor.Install, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	probers := supervisor.DefaultProbers(home, optDir)
	if appServer != "" {
		probers.Override = func() (supervisor.Install, bool) {
			return supervisor.OverrideInstall(appServer)
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

	slog.Info("zcode-acp: app-server resolved",
		"layout", inst.Layout, "binary", inst.Binary, "appimage", inst.AppImage, "version", inst.Version)

	return inst, nil
}

// startSupervised starts the supervisor over the SpawnFunc seam and waits
// for the first spawned child; the returned channel carries Start's outcome
// (nil on the intentional shutdown path). The spawn seam hands SpawnChild
// the Phase-3 canonical argv (SpawnArgv) and allowlisted env (ChildEnv —
// the proxy vars scrubbed by construction), under the SEPARATE spawn ctx
// whose cancellation is the post-teardown kill backstop (see run).
func startSupervised(
	supCtx context.Context, spawnCtx context.Context, supCancel context.CancelFunc,
	inst supervisor.Install, mount *supervisor.Mount,
) (*supervisor.Child, <-chan error, error) {
	spawnCh := make(chan *supervisor.Child, 1)
	startDone := make(chan error, 1)

	sup := supervisor.NewSupervisor(func(_ context.Context) (*supervisor.Child, error) {
		// The mount guard runs before EVERY spawn (the first one included): a
		// vanished root must fail loud, never respawn against it (MNT-05,
		// T-10-08). A nil mount is healthy.
		if guardErr := mount.Guard(); guardErr != nil {
			return nil, fmt.Errorf("appimage mount is not usable: %w", guardErr)
		}

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
