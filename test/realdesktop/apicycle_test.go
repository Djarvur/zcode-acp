//go:build realdesktop

// The guarded real-tier api-key FULL CYCLE (Phase 5 plan 05-09) — MANUAL-ONLY,
// never CI (the 03-07 discipline, internal/supervisor/realdesktop_test.go).
// The realdesktop build tag excludes this file from default builds, vet, and
// CI; the ZCODE_ACP_REAL_E2E=1 runtime guard skips the body even when someone
// builds with the tag (belt and braces), and a missing api key skips too —
// the key is read from the environment at run time, never embedded, never
// logged, never committed.
//
// Canonical run command (the END-OF-PHASE verification act — orchestrator-run
// with ledgered quota, never an executor step; the keyring must be unlocked
// and the key exported from the local ZCode desktop config into THIS shell
// only):
//
//	export ZCODE_ACP_API_KEY=<key from the local ZCode desktop config>  # never commit
//	export ZCODE_ACP_RAW_CAPTURE=<scratch path OUTSIDE this repo>       # raw captures are born outside
//	ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/
//
// Expected: PASS — authenticate{zai-api-key} proven by the hostsvc registry
// read-back verdict, one tiny tool-less turn, one file-read tool turn on a
// scratch file under the workspace, session/resume (same session, re-issued
// subscribe), session/close (clean teardown, post-close prompt rejected),
// staged shutdown with the zero-orphan proof. Without the env var AND the tag
// the test is skipped/absent — CI never contacts the desktop or the account.
//
// Quota note (D-05 minimum): planned spend is one registry read-back (no
// turn) + two tiny turns (one tool-less, one file-read) + the resume/close
// calls (no model spend). The Phase-5 quota-ledger row in
// experiments/README.md records the planned spend (PENDING) and takes the
// actuals from this run.
//
// Evidence discipline: when ZCODE_ACP_RAW_CAPTURE is set the raw capture is
// written UNSCRUBBED to that path and must pass the scrubber BEFORE any
// commit (Phase-1 pipeline; ground rule 1: raw captures are born outside the
// repo):
//
//	node experiments/scrub-capture.mjs <raw-capture> experiments/captures/EXP-APIKEY-CYCLE-01.jsonl --redact-path <workspace>
//
// The ledger entry then records the scrubbed capture + the V-EXP id
// (EXP-APIKEY-CYCLE-01), never credentials. Expected wire shape (A1):
// interaction/requestProviderRuntimeHeaders never fires on zai-api turns; if
// it does, the wired key source answers it (CORE-06) and the capture records
// the ask.

package realdesktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Manual-tier knobs: the double-guard envs, the optional override/env
// surface, and the deadline scale (Electron startup takes seconds; a real
// turn with the cipher ask fires ~10 s in and streams for tens of seconds —
// the deadline bounds a FULL degraded path at ~4 min per turn, never the
// happy path).
const (
	// realE2EEnv is the runtime guard: the body never runs without it.
	realE2EEnv = "ZCODE_ACP_REAL_E2E"

	// envAPIKey carries the z.ai api key (the D-05 config source — the same
	// var production parses; an empty value skips the test rather than
	// exercise the honest no-key error, which the composed tier locks).
	envAPIKey = "ZCODE_ACP_API_KEY"

	// envAppServer optionally overrides install discovery (the documented
	// order step 1 — the same override seam production exposes via
	// --app-server/ZCODE_ACP_APP_SERVER).
	envAppServer = "ZCODE_ACP_APP_SERVER"

	// envModel/envReasoning optionally override the model selection
	// (providerId/modelId), mirroring production parseConfig; the default
	// is the corpus-locked selection (the D-05 minimal-turn shape proven on
	// this account).
	envModel     = "ZCODE_ACP_MODEL"
	envReasoning = "ZCODE_ACP_REASONING_LEVEL"

	// envRawCapture optionally points the wire tee at a raw capture path
	// (born OUTSIDE the repo, per the Phase-1 ground rule 1). Unset = no
	// capture; the ledgered evidence run sets it.
	envRawCapture = "ZCODE_ACP_RAW_CAPTURE"

	// envWorkspace optionally overrides the scratch workspace directory
	// (default: a test temp dir; the override exists so the evidence run
	// can supply a stable --redact-path prefix for the scrubber).
	envWorkspace = "ZCODE_ACP_WORKSPACE"

	// optDir is the package-family discovery root (the deb layout).
	optDir = "/opt/ZCode"

	// secretToolBin is the libsecret CLI the D-06 credential loader shells
	// (a system binary on the real tier — never a Go module, T-5-SC).
	secretToolBin = "secret-tool"

	// captureID is the V-EXP id the evidence pipeline records (the ledger
	// entry names it; the scrubbed capture lands under experiments/captures/
	// under the same id).
	captureID = "EXP-APIKEY-CYCLE-01"

	// agentVersion is the honest agentInfo version (the production default
	// until the Phase-7 release flow stamps a real value).
	agentVersion = "0.0.0-dev"

	// spawnWait bounds the wait for the supervisor's first spawned child.
	spawnWait = 10 * time.Second

	// apiCycleDeadline bounds one full guarded pass (discover → spawn →
	// authenticate → two tiny turns → resume → close → staged shutdown).
	// The degraded path (locked keyring → the cipher ask's 180 s tracked
	// no-answer per turn) stays inside it; the happy path lands in minutes.
	apiCycleDeadline = 12 * time.Minute

	// shutdownWatchdog bounds OUR wait on the supervisor's staged teardown.
	shutdownWatchdog = 30 * time.Second

	// groupGoneDeadline/groupGonePoll drive the polled zero-orphan proof.
	groupGoneDeadline = 30 * time.Second
	groupGonePoll     = 100 * time.Millisecond
)

// Latch-drain knobs (plan 05-13): the desktop held its one-turn-at-a-time
// latch ~7.5 s post-ack in attempt 2 (EXP-APIKEY-CYCLE-02: send ack 45.010,
// refusal 52.514, cipher asks live until 52.462 — 50 ms before the refused
// send), so the drain is a wide-window wait by design. The totals stay well
// inside apiCycleDeadline's 12 min.
const (
	// askQuietSettle is the interaction-ask silence the settle phase waits
	// out before the second send (3 s quiet after the last interaction/*
	// ask — the desktop's cipher-ask loop ran at ~500 ms cadence and went
	// quiet only ~50 ms before attempt 2's refused send).
	askQuietSettle = 3 * time.Second

	// askSettleBudget bounds the settle phase (the ask loop must end
	// eventually; a pathological loop still proceeds — the retry phase
	// discriminates honestly).
	askSettleBudget = 30 * time.Second

	// settlePoll is the settle phase's re-check cadence.
	settlePoll = 250 * time.Millisecond

	// latchRetryStart/latchRetryMax are the drain's backoff band.
	latchRetryStart = 2 * time.Second
	latchRetryMax   = 15 * time.Second

	// latchRetryBudget/latchRetryAttempts bound the drain: a persistent
	// invalid-request refusal past this budget is hypothesis-(b) evidence
	// (premature end_turn), surfaced loudly for the attempt-3 verdict —
	// never masked by further retries.
	latchRetryBudget   = 120 * time.Second
	latchRetryAttempts = 8
)

// TestAPIKeyFullCycle runs the REAL guarded composition — the exact stack
// production run() builds (discovery → supervisor SpawnChild with the
// allowlist env → zcode.Conn over the exec pipes → the Front over the
// exported composition pieces → the full Responder policy table wired as
// conn.OnRequest BEFORE any traffic, cipher handler included, key sourced
// from the environment at run time), then the cycle: authenticate{zai-api-key}
// (the read-back verdict) → session/new → one tiny tool-less turn → one
// file-read tool turn on a scratch file → session/resume → session/close →
// staged shutdown. Assertions stay at the honest wire tier (see the leg
// helpers). MANUAL-ONLY (see the file header).
func TestAPIKeyFullCycle(t *testing.T) {
	if os.Getenv(realE2EEnv) != "1" {
		t.Skip("manual-only: set ZCODE_ACP_REAL_E2E=1 AND build with -tags realdesktop — CI never contacts the desktop or the account")
	}

	if os.Getenv(envAPIKey) == "" {
		t.Skip("no api key: export ZCODE_ACP_API_KEY from the local ZCode desktop config (never commit it, never log it)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), apiCycleDeadline)
	defer cancel()

	sel := resolveModelSelection()
	ws := workspaceDir(t)
	readTarget := writeScratchFile(t, ws)

	inst := discoverRealInstall(ctx, t)
	logObservedInstall(t, inst)

	capture := openCapture(t, inst)

	// Two lifecycles, deliberately (the production run() discipline): the
	// supervision ctx drives the supervisor's staged teardown, the spawn
	// ctx is the exec-level kill backstop cancelled only AFTER the staged
	// teardown completed — a shared cancellation would SIGKILL the child
	// mid-flight and the graceful stdin-EOF drain would never run.
	supCtx, supCancel := context.WithCancel(context.Background())
	spawnCtx, spawnCancel := context.WithCancel(context.Background())
	defer spawnCancel()

	child := startSupervisedChild(t, supCtx, spawnCtx, inst)

	//nolint:contextcheck // NewConn owns its base context (ctx only bounds the legs below)
	conn := zcode.NewConn(newTeeDuplex(capture, child))
	defer func() { _ = conn.Close() }() // the staged shutdown owns stdin from here

	// One key, one source (AUTH-01): the env-sourced binding feeds the
	// authenticate machinery, the runtime-headers answer, and the snapshot
	// builder alike — in-memory only (D-05).
	keys := hostsvc.NewStaticKey(os.Getenv(envAPIKey), "")

	front := buildCycleFront(conn, inst, keys, sel)

	// The permission sender binds BEFORE any frame flows (the acpport
	// binder discipline): this composition IS the ACP client, so the test
	// binds its own decision stand-in.
	front.BindPermissionSender(allowFirstOptionSender(t))

	// conn.OnRequest assigned BEFORE any traffic — the Conn hook contract
	// (production wireResponder order: responder policy table with the
	// permission bridge, the runtime-headers key source, and the D-06
	// cipher decrypt seam; the loader EXECUTES only when a real cipher ask
	// arrives — CI never runs this file). The ask tap wraps the assigned
	// chain immediately after, still before any frame flows: interaction/*
	// asks are recorded structurally (method + UTC timestamp, never params
	// — T-5-33) to drive the turn-2 latch drain.
	wireCycleResponder(conn, front, keys)
	asks := tapInteractionAsks(t, conn)

	authenticateLeg(t, ctx, front)
	sid := newSessionLeg(t, ctx, front, ws)
	toollessTurnLeg(t, ctx, front, sid)

	// The (a)/(b) discrimination anchor (plan 05-13): this UTC resolution
	// instant correlates 1:1 with the capture tee's ts entries — attempt 3's
	// evidence arbitrates desktop latch-lag (hypothesis a) vs premature
	// end_turn (hypothesis b). Turn 1's quiet floor runs 3x2500 ms from the
	// send ack (watcher.go PollConfig defaults), so the settle lands at
	// floor expiry when message activity goes quiet.
	t.Logf("turn 1 resolved at %s stop=end_turn — quiet-floor expiry per watcher.go floor math",
		time.Now().UTC().Format(time.RFC3339Nano))

	toolTurnLeg(t, ctx, front, asks, sid, readTarget)
	resumeLeg(t, ctx, front, sid, ws)
	closeLeg(t, ctx, front, sid)

	// Staged shutdown via the supervisor, then the zero-orphan proof.
	supCancel()
	waitStagedShutdown(t, child)
	spawnCancel() // the teardown backstop: the group is long gone; ESRCH-quiet

	assertGroupGone(t, child)

	capture.report(t)

	t.Log("full cycle complete on the real account: authenticate (read-back proven) → " +
		"new → tool-less turn → file-read tool turn → resume → close → staged shutdown")
}

// modelSelection mirrors acpfront.ModelSelection's vocabulary for the
// resolve step (providerId/modelId plus the optional reasoning level).
type modelSelection struct {
	ProviderID     string
	ModelID        string
	ReasoningLevel string
}

// resolveModelSelection defaults to the corpus-locked selection
// (zai-api/GLM-5.3-Flash + high — the D-05 minimal-turn shape proven on this
// account; the setModel follow-up carries the level because level-strict
// providers refuse the empty-options shape) and honors the production env
// surface: ZCODE_ACP_MODEL as providerId/modelId, ZCODE_ACP_REASONING_LEVEL
// for the level.
func resolveModelSelection() modelSelection {
	sel := modelSelection{ProviderID: "zai-api", ModelID: "GLM-5.3-Flash", ReasoningLevel: "high"}

	if raw := os.Getenv(envModel); raw != "" {
		if provider, model, ok := strings.Cut(raw, "/"); ok && provider != "" && model != "" {
			sel = modelSelection{ProviderID: provider, ModelID: model, ReasoningLevel: os.Getenv(envReasoning)}
		}
	}

	return sel
}

// workspaceDir resolves the scratch workspace: the env override (created,
// made absolute) or a test temp dir. The session/new cwd MUST be absolute
// (the front validates before any backend call).
func workspaceDir(t *testing.T) string {
	t.Helper()

	if ws := os.Getenv(envWorkspace); ws != "" {
		if err := os.MkdirAll(ws, 0o750); err != nil {
			t.Fatalf("create workspace dir %s: %v", ws, err)
		}

		abs, err := filepath.Abs(ws)
		if err != nil {
			t.Fatalf("resolve workspace abs path: %v", err)
		}

		return abs
	}

	return t.TempDir()
}

// writeScratchFile creates the file-read turn's target (a tiny three-line
// scratch file — no credential-shaped content) under the workspace.
func writeScratchFile(t *testing.T, ws string) string {
	t.Helper()

	path := filepath.Join(ws, "notes.txt")

	const content = "alpha line\nbeta line\nthird line\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write scratch file %s: %v", path, err)
	}

	return path
}

// discoverRealInstall runs the REAL discovery (the override seam first when
// ZCODE_ACP_APP_SERVER is set, the production matrix otherwise) and logs
// every probe row (the diagnostics ARE the product). A host without an
// install SKIPS — the full-cycle path needs one.
func discoverRealInstall(ctx context.Context, t *testing.T) supervisor.Install {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home for discovery: %v", err)
	}

	probers := supervisor.DefaultProbers(home, optDir)
	if override := os.Getenv(envAppServer); override != "" {
		probers.Override = func() (supervisor.Install, bool) {
			return overrideInstall(override), true
		}
	}

	inst, probes, err := probers.Discover(ctx)
	if err != nil {
		for _, row := range probes {
			t.Logf("probe %-16s %s -> %s", row.Name, row.Query, row.Result)
		}

		if errors.Is(err, supervisor.ErrNoInstall) {
			t.Skip("no ZCode desktop install found on this host — the manual full-cycle path needs one")
		}

		t.Fatalf("discover: %v", err)
	}

	return inst
}

// overrideInstall maps the override path into the discovery Install shape
// (main.go overrideInstall's exported-piece mirror: a directory validates
// against the invariant spawn surface; a regular file is the app-server
// binary itself).
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

// logObservedInstall records the probe-time install facts (capture-header
// discipline: record the layout and the builtin revision, never assume).
func logObservedInstall(t *testing.T, inst supervisor.Install) {
	t.Helper()

	revision, revErr := readBuiltinRevision(inst.BuiltinFile)
	if revErr != nil {
		t.Logf("observed install: layout=%s root=%s binary=%s builtinRevision=unknown (%v)",
			inst.Layout, inst.Root, inst.Binary, revErr)

		return
	}

	t.Logf("observed install: layout=%s root=%s binary=%s builtinRevision=%s",
		inst.Layout, inst.Root, inst.Binary, supervisor.BuiltinRevision(inst.BuiltinFile, revision))
}

// readBuiltinRevision reads the recorded revision number out of the active
// zcode-builtin.json (probe-time state — recorded, never assumed).
func readBuiltinRevision(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	var parsed struct {
		Revision int `json:"revision"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, err
	}

	return parsed.Revision, nil
}

// startSupervisedChild spawns the REAL app-server through the production
// path (SpawnChild + the allowlist ChildEnv) under the two-lifecycle
// discipline and hands back the first spawned child (race-safe delivery).
func startSupervisedChild(t *testing.T, supCtx, spawnCtx context.Context, inst supervisor.Install) *supervisor.Child {
	t.Helper()

	spawnCh := make(chan *supervisor.Child, 1)

	sup := supervisor.NewSupervisor(func(_ context.Context) (*supervisor.Child, error) {
		child, err := supervisor.SpawnChild(spawnCtx,
			supervisor.SpawnArgv(inst),
			supervisor.ChildEnv(inst, os.Environ()),
			supervisor.SpawnOpts{})
		if err == nil {
			spawnCh <- child
		}

		return child, err
	}, supervisor.DefaultPolicy())

	go func() { _ = sup.Start(supCtx) }()

	return waitSpawnedChild(t, spawnCh)
}

// waitSpawnedChild receives the SpawnFunc's child delivery.
func waitSpawnedChild(t *testing.T, spawnCh <-chan *supervisor.Child) *supervisor.Child {
	t.Helper()

	select {
	case child := <-spawnCh:
		return child
	case <-time.After(spawnWait):
		t.Fatalf("no child delivered by the spawn path within %v", spawnWait)

		return nil
	}
}

// childDuplex joins the child's stdout read side and stdin write side into
// the one io.ReadWriteCloser the codec Conn consumes (the 03-07 shape).
// Close unblocks a blocked Read, which is what Conn.Close relies on.
type childDuplex struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

// Read implements io.Reader over the child's stdout.
func (d childDuplex) Read(p []byte) (int, error) {
	//nolint:wrapcheck // the transport passthrough IS the contract (the codec Conn owns error interpretation)
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

// teeDuplex wraps the child pipes with the wire tee — both directions are
// recorded (the Phase-1 capture discipline: host→server and server→host
// alike) before the evidence pipeline scrubs them.
type teeDuplex struct {
	inner   childDuplex
	capture *captureTee
}

// newTeeDuplex joins the child's pipes under the capture tee.
func newTeeDuplex(capture *captureTee, child *supervisor.Child) teeDuplex {
	return teeDuplex{
		inner:   childDuplex{reader: child.Stdout, writer: child.Stdin},
		capture: capture,
	}
}

// Read forwards the child's stdout and records what flowed.
func (d teeDuplex) Read(p []byte) (int, error) {
	n, err := d.inner.Read(p)
	if n > 0 {
		d.capture.record("stdout", p[:n])
	}

	//nolint:wrapcheck // the transport passthrough IS the contract (see childDuplex.Read)
	return n, err
}

// Write forwards to the child's stdin and records what flowed.
func (d teeDuplex) Write(p []byte) (int, error) {
	n, err := d.inner.Write(p)
	if n > 0 {
		d.capture.record("stdin", p[:n])
	}

	//nolint:wrapcheck // the transport passthrough IS the contract (see childDuplex.Write)
	return n, err
}

// Close ends both pipe halves.
func (d teeDuplex) Close() error {
	return d.inner.Close()
}

// captureTee is the raw-capture writer (the ZCODE_ACP_RAW_CAPTURE path):
// line 1 is the Phase-1 capture header, every following line records one
// wire chunk as JSON ({ts, dir, line} — the probe's JSON.stringify({ts,
// line}) shape plus the direction). The file stays UNSCRUBBED — the
// scrubber is the only path into experiments/captures/.
type captureTee struct {
	mu  sync.Mutex
	out *os.File
	bad bool
}

// openCapture opens the raw capture when the env path is set and writes the
// header; unset leaves the run uncaptured (the ledgered evidence run sets
// it).
func openCapture(t *testing.T, inst supervisor.Install) *captureTee {
	t.Helper()

	path := os.Getenv(envRawCapture)
	if path == "" {
		t.Log("no raw capture path set (ZCODE_ACP_RAW_CAPTURE) — the wire is not recorded; the ledgered evidence run sets it")

		return nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open raw capture %s: %v", path, err)
	}

	t.Cleanup(func() { _ = f.Close() })

	header, err := json.Marshal(map[string]string{
		"capture":         captureID,
		"date":            time.Now().UTC().Format(time.RFC3339),
		"desktopVersion":  "unknown", // recorded, never assumed (WR-04)
		"builtinRevision": builtinRevisionLabel(inst),
		"host":            runtime.GOOS + "/" + runtime.GOARCH,
	})
	if err != nil {
		t.Fatalf("marshal capture header: %v", err)
	}

	if _, err := f.Write(append(header, '\n')); err != nil {
		t.Fatalf("write capture header: %v", err)
	}

	return &captureTee{out: f}
}

// builtinRevisionLabel renders the observed revision formula (or "unknown"
// when the builtin file is unreadable — recorded, never assumed).
func builtinRevisionLabel(inst supervisor.Install) string {
	revision, err := readBuiltinRevision(inst.BuiltinFile)
	if err != nil {
		return "unknown"
	}

	return supervisor.BuiltinRevision(inst.BuiltinFile, revision)
}

// record appends one wire chunk (best-effort: a mid-run failure flips bad
// once and drops the rest — a capture problem must never corrupt the run's
// verdict).
func (c *captureTee) record(dir string, p []byte) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.bad {
		return
	}

	entry, err := json.Marshal(map[string]string{
		"ts":   time.Now().UTC().Format(time.RFC3339Nano),
		"dir":  dir,
		"line": string(p),
	})
	if err != nil {
		c.bad = true

		return
	}

	if _, err := c.out.Write(append(entry, '\n')); err != nil {
		c.bad = true
	}
}

// report reminds the runner of the scrub-first evidence step (D-05: the
// raw capture passes scrub-capture.mjs before any commit; the ledger entry
// records the scrubbed capture + the V-EXP id, never credentials).
func (c *captureTee) report(t *testing.T) {
	t.Helper()

	if c == nil {
		return
	}

	t.Logf("raw capture written (UNSCRUBBED — scrub before any commit): "+
		"node experiments/scrub-capture.mjs <$%s> experiments/captures/%s.jsonl --redact-path <workspace>",
		envRawCapture, captureID)
}

// allowFirstOptionSender is the composition's ACP-client stand-in for the
// permission bridge (production binds the SDK connection's sender at
// construction; this composition IS the client). It answers the first
// allow-class option — the same allow_once decision the composed-tier cycle
// drives — falling back to the first option, and logs the ask structurally
// (tool name only, never params).
func allowFirstOptionSender(t *testing.T) acpport.PermissionSender {
	return func(ctx context.Context, ask acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
		_ = ctx

		t.Logf("permission ask: session=%s tool=%s options=%d", ask.SessionID, ask.ToolName, len(ask.Options))

		for _, opt := range ask.Options {
			if opt.Kind == acpport.PermKindAllowOnce || opt.Kind == acpport.PermKindAllowAlways {
				return acpport.PermissionOutcome{OptionID: opt.OptionID}, nil
			}
		}

		if len(ask.Options) > 0 {
			return acpport.PermissionOutcome{OptionID: ask.Options[0].OptionID}, nil
		}

		return acpport.PermissionOutcome{Cancelled: true}, nil
	}
}

// buildCycleFront constructs the Front over the exported composition pieces,
// mirroring production buildFront: the agentInfo version, the model
// selection (the corpus-locked default or the env override), and the
// AUTH-01 authenticate machinery (the key binding, the resolved install for
// the snapshot's exact-revision builtin formula, the read-back want list —
// the configured model, mirroring production readbackModels).
func buildCycleFront(
	conn *zcode.Conn, inst supervisor.Install, keys hostsvc.KeySource, sel modelSelection,
) *acpfront.Front {
	return acpfront.New(conn,
		acpfront.WithVersion(agentVersion),
		acpfront.WithModel(acpfront.ModelSelection{
			ProviderID:     sel.ProviderID,
			ModelID:        sel.ModelID,
			ReasoningLevel: sel.ReasoningLevel,
		}),
		acpfront.WithAuth(acpfront.AuthConfig{
			Keys:       keys,
			Install:    inst,
			WantModels: []string{sel.ProviderID + "/" + sel.ModelID},
		}),
	)
}

// wireCycleResponder mirrors production wireResponder verbatim (via the
// exported composition pieces): the full D-01 policy table — the 05-02
// permission bridge, the 05-06 runtime-headers key source, the D-06
// credential-decrypt seam (the loader EXECUTES only when a real cipher ask
// arrives; the key it needs is the USER'S OWN login-keyring material, never
// ours) — assigned as conn.OnRequest BEFORE any traffic.
func wireCycleResponder(conn *zcode.Conn, front *acpfront.Front, keys hostsvc.KeySource) {
	responder := interact.NewResponder(
		interact.WithPermissionForwarder(interact.NewPermissionBridge(front.PermissionForward()).Forward),
		interact.WithRuntimeHeadersKeySource(hostsvc.RuntimeHeadersHandler(keys)),
		interact.WithCredentialDecrypt(interact.NewCredentialDecrypt(
			interact.NewSecretToolKeyLoader(secretToolBin, credentialsPath(), interact.ScrubbedEnv(os.Environ())))),
	)

	conn.OnRequest = responder.Serve
}

// tapInteractionAsks wraps the ALREADY-ASSIGNED conn.OnRequest with the ask
// recorder: every interaction/* ask is recorded structurally (method + UTC
// timestamp, never params — T-5-33) into the returned recorder — the settle
// signal the turn-2 latch drain waits on. Call it before any traffic (the
// Conn hook contract); the wrapped chain falls through to the full
// responder untouched.
func tapInteractionAsks(t *testing.T, conn *zcode.Conn) *askRecorder {
	t.Helper()

	next := conn.OnRequest
	if next == nil {
		t.Fatal("tapInteractionAsks: wireCycleResponder must assign conn.OnRequest before the tap wraps it")
	}

	tap := &askRecorder{}

	conn.OnRequest = func(reqCtx context.Context, env zcode.Envelope) (any, *zcode.WireError, bool) {
		if strings.HasPrefix(env.Method, "interaction/") {
			now := time.Now().UTC()

			tap.mu.Lock()
			tap.lastAsk = now
			tap.mu.Unlock()

			// Structural logging ONLY (conn.go:436-450 precedent): the
			// cipher asks' params carry the encrypted credential value
			// verbatim — method name + UTC timestamp, never params (T-5-33).
			t.Logf("interaction ask: %s at %s", env.Method, now.Format(time.RFC3339Nano))
		}

		return next(reqCtx, env)
	}

	return tap
}

// askRecorder is the interaction-ask tap's state: the UTC instant of the
// most recent interaction/* request (the desktop's cipher-ask loop is the
// latch's live signal — EXP-APIKEY-CYCLE-02 asks ran until 52.462, 50 ms
// before the refused send). Guarded by mu: request handlers are concurrent
// on the conn's read goroutine fan-out.
type askRecorder struct {
	mu      sync.Mutex
	lastAsk time.Time
}

// lastAskAt returns the UTC instant of the most recent interaction/* ask
// (zero time when none was seen — the drain settles immediately).
func (r *askRecorder) lastAskAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.lastAsk
}

// credentialsPath is the desktop credentials file the login-keyring key is
// bound to (the field-verified sha256-realpath derivation input — the
// production main.go derivation; missing home yields an empty path and the
// cipher ask degrades honestly, D-06).
func credentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".zcode", "v2", "credentials.json")
}

// noteCollector collects the Front's UpdateSink notes per turn (the honest
// wire tier: every note traces to one received backend event).
type noteCollector struct {
	mu    sync.Mutex
	notes []acpport.SessionNote
}

// sink implements acpport.UpdateSink.
func (c *noteCollector) sink(_ context.Context, note acpport.SessionNote) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.notes = append(c.notes, note)

	return nil
}

// toolNotes returns the tool-call lifecycle frames the turn observed.
func (c *noteCollector) toolNotes() []acpport.ToolNote {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]acpport.ToolNote, 0, len(c.notes))
	for _, note := range c.notes {
		if note.Kind == acpport.NoteTool && note.Tool != nil {
			out = append(out, *note.Tool)
		}
	}

	return out
}

// authenticateLeg runs authenticate{zai-api-key} and asserts the honest
// read-back verdict: Front.Authenticate returns nil ONLY after the registry
// read-back confirms the want models (the PITFALL-1 chain: snapshot push →
// refresh → read-back; never the received ack).
func authenticateLeg(t *testing.T, ctx context.Context, front *acpfront.Front) {
	t.Helper()

	if err := front.Authenticate(ctx, acpfront.AuthMethodZaiAPIKey); err != nil {
		t.Fatalf("authenticate{zai-api-key}: %v", err)
	}

	t.Log("authenticate ok — the registry read-back verdict proved the key landed (PITFALL 1: the verdict is the registry's)")
}

// newSessionLeg opens the session over the scratch workspace. The
// 06-04 configOptions return (the neutral option list built from the
// session's confirmed settings snapshot) is logged structurally — the
// api-key cycle's assertion surface stays the read-back + turn legs.
func newSessionLeg(t *testing.T, ctx context.Context, front *acpfront.Front, ws string) string {
	t.Helper()

	sid, options, err := front.NewSession(ctx, acpport.NewSessionParams{Cwd: ws})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	for i := range options {
		t.Logf("configOption: id=%s currentValue=%q entries=%d", options[i].ID, options[i].CurrentValue, len(options[i].Options))
	}

	t.Logf("session/new ok: %s (create + the D-03 subscribe gate%s, %d configOptions)", sid, setModelNote(t), len(options))

	return sid
}

// setModelNote records which session/new preamble shape ran (the setModel
// follow-up rides only when a model is configured — it always is here).
func setModelNote(t *testing.T) string {
	t.Helper()

	return " + setModel " + resolveModelSelection().ProviderID + "/" + resolveModelSelection().ModelID
}

// toollessTurnLeg runs the tiny tool-less turn and asserts the honest
// outcome: end_turn, ZERO tool_call frames (the definition of tool-less —
// a model that calls a tool anyway fails the leg honestly).
func toollessTurnLeg(t *testing.T, ctx context.Context, front *acpfront.Front, sid string) {
	t.Helper()

	col := &noteCollector{}

	stop, err := front.Prompt(ctx, acpport.PromptParams{
		SessionID: sid,
		Text:      "Reply with the single word: ok",
	}, col.sink)
	if err != nil {
		t.Fatalf("tool-less turn: %v", err)
	}

	if stop != acpport.StopEndTurn {
		t.Fatalf("tool-less turn stop reason: got %q, want end_turn", stop)
	}

	if tools := col.toolNotes(); len(tools) != 0 {
		t.Fatalf("tool-less turn observed %d tool_call frames; want zero", len(tools))
	}

	t.Log("tiny tool-less turn ok: end_turn, zero tool frames (the cipher ask answered mid-turn if it fired)")
}

// turnWithLatchDrain runs one prompt turn through the desktop's
// one-turn-at-a-time latch (plan 05-13). Phase settle: wait for
// interaction-ask silence (askQuietSettle quiet after the last
// interaction/* ask, bounded by askSettleBudget) — the desktop's cipher-ask
// loop was live until ~50 ms before attempt 2's refused send, so an
// immediate second send races the lingering latch; no ask ever seen settles
// immediately. Phase retry: front.Prompt retried ONLY on the invalid-request
// class (T-5-32: BOTH concurrency surfaces — the front-wide slot guard and
// the machine's -32010 refusal — map onto ONE acpport invalid-request error
// carrying turnActiveMessage, front.go Prompt; the CODE is matched via
// errors.As, never the message text). Any other error — including a
// *acpport.Error with a different code — fails the leg immediately; a drain
// that retried -32603 internals would green-wash a real defect. A refusal
// persisting past the budget returns the hypothesis-(b) sentinel for the
// attempt-3 verdict, loudly.
func turnWithLatchDrain(
	t *testing.T, ctx context.Context, front *acpfront.Front, asks *askRecorder,
	sid, prompt string, sink acpport.UpdateSink,
) (acpport.StopReason, error) {
	t.Helper()

	// Phase settle: the latch is held while the ask loop is live — wait for
	// its silence before the send.
	settleDeadline := time.Now().Add(askSettleBudget)

	for {
		last := asks.lastAskAt()

		if last.IsZero() {
			t.Logf("latch drain settle: no interaction ask seen — settling immediately at %s",
				time.Now().UTC().Format(time.RFC3339Nano))

			break
		}

		quiet := time.Since(last)
		if quiet >= askQuietSettle {
			t.Logf("latch drain settle: %s of ask silence at %s (last ask %s)",
				quiet.Truncate(time.Millisecond), time.Now().UTC().Format(time.RFC3339Nano),
				last.Format(time.RFC3339Nano))

			break
		}

		if time.Now().After(settleDeadline) {
			t.Logf("latch drain settle: %s budget exhausted at %s (last ask %s) — proceeding to the retry phase",
				askSettleBudget, time.Now().UTC().Format(time.RFC3339Nano), last.Format(time.RFC3339Nano))

			break
		}

		time.Sleep(settlePoll)
	}

	// Phase retry: bounded invalid-request-class drain.
	backoff := latchRetryStart
	started := time.Now()

	for attempt := 1; ; attempt++ {
		stop, err := front.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: prompt}, sink)
		if err == nil {
			t.Logf("latch drain: prompt accepted at %s (attempt %d)", time.Now().UTC().Format(time.RFC3339Nano), attempt)

			return stop, nil
		}

		var acpErr *acpport.Error
		if !errors.As(err, &acpErr) || acpErr.Code != acpport.CodeInvalidRequest {
			// T-5-32: never retry outside the invalid-request class —
			// anything else (including -32603 internals) is a genuine
			// defect the drain must surface, not mask.
			return "", err
		}

		t.Logf("latch refusal %d/%d at %s (invalid-request class): %s — draining",
			attempt, latchRetryAttempts, time.Now().UTC().Format(time.RFC3339Nano), acpErr.Message)

		if attempt >= latchRetryAttempts || time.Since(started)+backoff > latchRetryBudget {
			return "", fmt.Errorf(
				"latch drain exhausted after %d attempts / %s budget: the invalid-request refusal persisted "+
					"past the drain budget (latchRetryBudget=%s, latchRetryAttempts=%d) — a persistent refusal is "+
					"hypothesis-(b) evidence (our end_turn fired before the desktop's true completion) for the "+
					"attempt-3 verdict; record it for the next gap plan, do not widen this drain",
				attempt, time.Since(started).Truncate(time.Millisecond), latchRetryBudget, latchRetryAttempts)
		}

		time.Sleep(backoff)

		backoff *= 2
		if backoff > latchRetryMax {
			backoff = latchRetryMax
		}
	}
}

// toolTurnLeg runs the file-read tool turn on the scratch file and asserts
// the honest wire tier: end_turn AND observed tool_call frames with a
// terminal status (start → completed/failed). Every observed frame is
// logged for the capture discipline — never an over-fit to exact titles.
// The prompt rides the latch drain (plan 05-13): this turn is the SECOND
// send the desktop's lingering latch refused in attempt 2
// (EXP-APIKEY-CYCLE-02.jsonl:118-119).
func toolTurnLeg(t *testing.T, ctx context.Context, front *acpfront.Front, asks *askRecorder, sid, filePath string) {
	t.Helper()

	col := &noteCollector{}

	prompt := fmt.Sprintf("Use the file-read tool to read %s, then reply with exactly: READ-OK", filePath)

	stop, err := turnWithLatchDrain(t, ctx, front, asks, sid, prompt, col.sink)
	if err != nil {
		t.Fatalf("file-read tool turn: %v", err)
	}

	if stop != acpport.StopEndTurn {
		t.Fatalf("file-read tool turn stop reason: got %q, want end_turn", stop)
	}

	tools := col.toolNotes()
	if len(tools) == 0 {
		t.Fatal("file-read tool turn observed ZERO tool_call frames — the tool leg was not exercised")
	}

	starts, terminals := 0, 0
	for _, tn := range tools {
		t.Logf("tool frame: id=%s start=%v name=%s kind=%s status=%s",
			tn.ToolCallID, tn.IsStart, tn.Name, tn.Kind, tn.Status)

		if tn.IsStart {
			starts++
		}

		if tn.Status == acpport.ToolStatusCompleted || tn.Status == acpport.ToolStatusFailed {
			terminals++
		}
	}

	if starts == 0 || terminals == 0 {
		t.Fatalf("file-read tool turn frames incomplete: %d starts, %d terminal updates over %d frames",
			starts, terminals, len(tools))
	}

	t.Log("file-read tool turn ok: tool_call frames observed with a terminal status (TOOL-01 live)")
}

// resumeLeg reattaches the same backend session (SESS-01): the returned id
// must be the SAME session (reattach, not a new session) and nil error
// proves the re-issued desktop-continuous subscribe was accepted (the D-03
// gate applies to a reattached session exactly as to a new one). The
// workspace cwd rides the widened seam (plan 05-10 WR-02 — the SDK
// ResumeSessionRequest.Cwd contract makes it MUST-be-absolute; the front
// validates and records it so post-resume tool paths resolve absolute).
// No replay is fabricated by construction — the client sees only future
// turns.
func resumeLeg(t *testing.T, ctx context.Context, front *acpfront.Front, sid string, cwd string) {
	t.Helper()

	resumed, err := front.ResumeSession(ctx, sid, cwd)
	if err != nil {
		t.Fatalf("session/resume: %v", err)
	}

	if resumed != sid {
		t.Fatalf("session/resume reattached %q; want the same session %q", resumed, sid)
	}

	t.Log("session/resume ok: same session reattached, the re-issued subscribe accepted (no replay)")
}

// closeLeg runs the ordered teardown (SESS-01) and proves it clean at the
// front tier: CloseSession returns nil, and a post-close prompt is rejected
// coded-clean BEFORE any backend call (the registry entry is released —
// never a zombie session; the rejection is local, no wire spend).
func closeLeg(t *testing.T, ctx context.Context, front *acpfront.Front, sid string) {
	t.Helper()

	if err := front.CloseSession(ctx, sid); err != nil {
		t.Fatalf("session/close: %v", err)
	}

	postCloseSink := func(_ context.Context, _ acpport.SessionNote) error { return nil }

	if _, err := front.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: "post-close"}, postCloseSink); err == nil {
		t.Fatal("post-close prompt was accepted — the close did not release the session")
	}

	t.Log("session/close ok: teardown clean, post-close prompt rejected coded-clean")
}

// waitStagedShutdown waits for the supervisor's staged teardown (driven by
// the supervision cancel) to complete.
func waitStagedShutdown(t *testing.T, child *supervisor.Child) {
	t.Helper()

	select {
	case <-child.Done:
	case <-time.After(shutdownWatchdog):
		t.Fatalf("the staged shutdown did not complete within %v", shutdownWatchdog)
	}
}

// assertGroupGone polls kill(-pgid, 0) until ESRCH — the zero-orphan proof
// (zombies exist until reaped, so the check is polled, never one racy call).
func assertGroupGone(t *testing.T, child *supervisor.Child) {
	t.Helper()

	deadline := time.Now().Add(groupGoneDeadline)

	for {
		err := syscall.Kill(-child.PgID, 0)
		if errors.Is(err, syscall.ESRCH) {
			t.Log("no orphaned group members: kill(-pgid, 0) reports ESRCH after the staged shutdown")

			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("the real desktop process group is still alive after the staged shutdown: kill(%d, 0) = %v",
				-child.PgID, err)
		}

		time.Sleep(groupGonePoll)
	}
}
