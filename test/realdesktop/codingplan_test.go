//go:build realdesktop

// The guarded real-tier coding-plan FULL CYCLE (Phase 6 plan 06-08, D-12) —
// MANUAL-ONLY, never CI (the 03-07 discipline; the apicycle sibling's guard
// triple copied verbatim). The realdesktop build tag excludes this file from
// default builds, vet, and CI; the ZCODE_ACP_REAL_E2E=1 runtime guard skips
// the body even when someone builds with the tag (belt and braces), and a
// desktop without the individual coding-plan login skips too — the desktop's
// keyring login IS the credential (D-12): NO key is exported, embedded,
// logged, or committed anywhere in this path.
//
// Canonical run command (the END-OF-PHASE verification act — HUMAN-GATED per
// D-12, the 05-12/05-14 discipline; orchestrator-run with ledgered quota,
// never an executor step; the desktop must be logged into the individual
// coding plan and the session's keyring unlocked):
//
//	export ZCODE_ACP_RAW_CAPTURE=<scratch path outside this repo>
//	ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/
//
// Expected: PASS — authenticate{zai-individual-coding-plan} proven by the
// registry read-back (account models present — never the ack), the real PAT
// minted via the official api.z.ai chain (in-process only, never persisted),
// session/new configOptions carrying both entries from the live registry
// (model mandatory with the confirmed currentValue; thought_level recorded
// either way — A4/A7's designed confirmer), ONE streamed tool-less tiny turn
// with the runtime-headers ask answered (token+id+accountScope), staged
// shutdown with the zero-orphan proof. Without the env var AND the tag the
// test is skipped/absent — CI never contacts the desktop or the account.
//
// Quota note (D-13 cap): planned spend is the real PAT mint + the registry
// read-back (registry-level, no model spend) + ONE tool-less tiny turn —
// nothing else. The quota-ledger row in experiments/README.md records the
// planned spend (PENDING) and takes the actuals from this run.
//
// Evidence discipline: when ZCODE_ACP_RAW_CAPTURE is set the raw capture is
// written UNSCRUBBED to that path and must pass the scrubber BEFORE any
// commit (Phase-1 pipeline; ground rule 1: raw captures are born outside the
// repo — T-06-24: the PAT/token material rides the wire bytes):
//
//	node experiments/scrub-capture.mjs <$ZCODE_ACP_RAW_CAPTURE> experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl --redact-path <workspace>
//
// The ledger entry then records the scrubbed capture + the V-EXP id
// (EXP-CODINGPLAN-CYCLE-01), never credentials; scan-secrets must stay green
// on the tree AND the staged diff; a failed leg is recorded honestly (the
// 05-14 pattern) and the phase stays open.

package realdesktop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Coding-plan tier knobs (the apicycle sibling's block, trimmed to this
// cycle): the capture id is THIS plan's V-EXP id and the deadline bounds
// one guarded pass (discover -> spawn -> authenticate -> session/new -> one
// tiny tool-less turn -> staged shutdown). Reused apicycle identifiers
// (realE2EEnv, the env surface, optDir, secretToolBin, spawnWait,
// shutdownWatchdog, the zero-orphan knobs, workspaceDir,
// discoverRealInstall, startSupervisedChild, the tee/capture duplex pair,
// assertGroupGone, waitStagedShutdown, noteCollector, credentialsPath) come
// from apicycle_test.go — one package, one definition per value.
const (
	// cpCaptureID is the V-EXP id the evidence pipeline records (the ledger
	// row names it; the scrubbed capture lands under experiments/captures/
	// under the same id).
	cpCaptureID = "EXP-CODINGPLAN-CYCLE-01"

	// cpCycleDeadline bounds one full guarded pass (Electron startup takes
	// seconds; the mint chain + one tiny turn land in minutes — the
	// apicycle scale, never the happy path's measure).
	cpCycleDeadline = 12 * time.Minute

	// cpOptionIDModel/cpOptionIDThoughtLevel are the configOptions wire ids
	// (BuildConfigOptions' table — the picker's own vocabulary).
	cpOptionIDModel        = "model"
	cpOptionIDThoughtLevel = "thought_level"
)

// TestCodingPlanFullCycle runs the REAL guarded coding-plan composition —
// the exact stack production run() builds (discovery → supervisor SpawnChild
// with the allowlist env → zcode.Conn over the exec pipes → the coding-plan
// machinery + Front over the exported composition pieces → the full
// Responder policy table wired as conn.OnRequest BEFORE any traffic: the
// permission bridge, the mode-aware PAT runtime-headers source over the
// SAME minter the front minted through, the D-06 cipher decrypt seam — every
// seam EXECUTES only when its ask arrives), then the cycle:
// authenticate{zai-individual-coding-plan} (the read-back verdict, the
// mint-once) → session/new (live-registry configOptions) → ONE tiny
// tool-less turn (the runtime-headers ask answered token+id+accountScope) →
// staged shutdown with the zero-orphan proof. Assertions read registry/live
// state, never acks. MANUAL-ONLY (see the file header).
func TestCodingPlanFullCycle(t *testing.T) {
	if os.Getenv(realE2EEnv) != "1" {
		t.Skip("manual-only: set ZCODE_ACP_REAL_E2E=1 AND build with -tags realdesktop — " +
			"CI never contacts the desktop or the account")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cpCycleDeadline)
	defer cancel()

	ws := workspaceDir(t)

	inst := discoverRealInstall(ctx, t)
	logObservedInstall(t, inst)

	// Skip-when-unconfigured (the honest pre-flight): the D-02 proactive
	// keyring probe over the REAL machine — no desktop login (or a locked
	// keyring) skips the run with the actionable hint; the real account is
	// never touched by a half-configured machine.
	prober := cpBuildMachinery(inst).prober
	if err := prober(ctx); err != nil {
		t.Skipf("the desktop coding-plan login is not reachable: %v — log in via the "+
			"ZCode desktop and unlock the keyring, then re-run (D-02's actionable posture)", err)
	}

	capture := cpOpenCapture(t, inst)

	// Two lifecycles, deliberately (the production run() discipline — the
	// apicycle comment holds verbatim): the supervision ctx drives the
	// staged teardown, the spawn ctx is the exec-level kill backstop.
	supCtx, supCancel := context.WithCancel(context.Background())
	spawnCtx, spawnCancel := context.WithCancel(context.Background())
	defer spawnCancel()

	child := startSupervisedChild(t, supCtx, spawnCtx, inst)

	//nolint:contextcheck // NewConn owns its base context (ctx only bounds the legs below)
	conn := zcode.NewConn(newTeeDuplex(capture, child))
	defer func() { _ = conn.Close() }() // the staged shutdown owns stdin from here

	// The production coding-plan machinery and the Front over it —
	// buildFront's exported-piece mirror (the coding-plan run carries no
	// api key: the coding-plan arm never reads the static source).
	machinery := cpBuildMachinery(inst)
	keys := hostsvc.NewStaticKey("", "")
	front := cpBuildFront(conn, inst, machinery, keys)

	// The permission sender binds BEFORE any frame flows (the acpport binder
	// discipline): this composition IS the ACP client.
	front.BindPermissionSender(allowFirstOptionSender(t))

	// conn.OnRequest assigned BEFORE any traffic — the Conn hook contract —
	// then the ask tap wraps the chain (still before any frame flows):
	// interaction/* asks recorded structurally (method + UTC timestamp,
	// never params — T-5-33) and every runtime-headers ANSWER recorded as a
	// SHAPE (field-presence booleans — never the token material itself).
	wireCodingPlanResponder(conn, front, machinery, keys)
	asks := cpTapInteractionAsks(t, conn)

	cpAuthenticateLeg(t, ctx, front)

	sid := cpSessionNewLeg(t, ctx, front, ws)

	cpToollessTurnLeg(t, ctx, front, asks, sid)

	// Staged shutdown via the supervisor, then the zero-orphan proof.
	supCancel()
	waitStagedShutdown(t, child)
	spawnCancel() // the teardown backstop: the group is long gone; ESRCH-quiet

	assertGroupGone(t, child)

	capture.report(t)

	t.Log("full coding-plan cycle complete on the real account: authenticate (read-back proven) -> " +
		"new (live configOptions) -> tool-less turn -> staged shutdown")
}

// settingPath is the desktop setting.json the production entitlement gate
// reads (main.go settingPath's mirror; missing home yields an empty path —
// the gate's missing-setting arm degrades honestly, never a phantom read).
func settingPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".zcode", "v2", "setting.json")
}

// cpDesktopVersionPattern/cpDesktopVersionFallback/cpClientVersion are
// main.go's best-effort desktop-version dig (the mint chain's clientVersion
// seam input) — defined beside cpBuildMachinery below.

// cpMachinery is the composed coding-plan machinery handle the test carries
// (the main.go codingPlanMachinery shape: store, prober, gate, minter).
type cpMachinery struct {
	creds  *interact.CredentialStore
	prober interact.KeyringProber
	gate   *interact.EntitlementGate
	minter *hostsvc.PatMinter
}

// cpBuildMachinery constructs the production coding-plan machinery over the
// real seams (main.go buildCodingPlanMachinery's mirror): the 06-01
// credential store (the v2 decrypt loader over the SAME secret-tool binary +
// credentials path the cipher seam uses), the D-02 proactive keyring prober,
// the production entitlement gate over settingPath, and the 06-02 PAT minter
// (the desktop-version clientVersion seam). Construction performs no I/O —
// every member executes lazily (D-13: nothing runs until its ask arrives).
func cpBuildMachinery(inst supervisor.Install) *cpMachinery {
	env := interact.ScrubbedEnv(os.Environ())

	creds := interact.NewCredentialStore(credentialsPath(),
		interact.NewSecretToolKeyLoader(secretToolBin, credentialsPath(), env))

	return &cpMachinery{
		creds:  creds,
		prober: interact.NewKeyringProber(secretToolBin, credentialsPath(), env),
		gate:   interact.NewEntitlementGate(settingPath(), creds),
		minter: hostsvc.NewPatMinter(
			hostsvc.WithPatClientVersion(cpClientVersion(inst)),
			hostsvc.WithPatCredentials(creds),
		),
	}
}

// cpBuildFront constructs the Front over the machinery (main.go buildFront's
// mirror): the honest agentInfo version, NO model selection by default (the
// session binds the backend's confirmed current — the env override rides
// ZCODE_ACP_MODEL/ZCODE_ACP_REASONING_LEVEL as in production), and the
// AuthConfig carrying BOTH auth paths' machinery (the static key source
// stays wired empty; the coding-plan seams carry the credential).
func cpBuildFront(
	conn *zcode.Conn, inst supervisor.Install, machinery *cpMachinery, keys hostsvc.KeySource,
) *acpfront.Front {
	opts := []acpfront.Option{acpfront.WithVersion(agentVersion)}

	if sel := cpResolveModelSelection(); sel != nil {
		opts = append(opts, acpfront.WithModel(acpfront.ModelSelection{
			ProviderID:     sel.ProviderID,
			ModelID:        sel.ModelID,
			ReasoningLevel: sel.ReasoningLevel,
		}))
	}

	opts = append(opts, acpfront.WithAuth(acpfront.AuthConfig{
		Keys:       keys,
		Install:    inst,
		WantModels: cpReadbackModels(),
		Creds:      machinery.creds,
		Pat:        machinery.minter,
		Probe:      machinery.prober,
		Entitle:    machinery.gate.Entitled,
	}))

	return acpfront.New(conn, opts...)
}

// cpResolveModelSelection honors the production env override; the default is
// NO selection (the session binds the backend's confirmed current — the
// D-13-minimal shape; a configured model adds the setModel follow-up and
// must carry the provider-qualified id).
func cpResolveModelSelection() *modelSelection {
	raw := os.Getenv(envModel)
	if raw == "" {
		return nil
	}

	provider, model, ok := strings.Cut(raw, "/")
	if !ok || provider == "" || model == "" {
		return nil
	}

	return &modelSelection{ProviderID: provider, ModelID: model, ReasoningLevel: os.Getenv(envReasoning)}
}

// cpReadbackModels is the api-key arm's documented catalog pair (the
// coding-plan arm derives ITS want list from the built overlay — this list
// rides AuthConfig parity only).
func cpReadbackModels() []string {
	return []string{"zai-api/GLM-5.3-Flash", "zai-api/GLM-5.3"}
}

// wireCodingPlanResponder mirrors production wireResponder verbatim (via the
// exported composition pieces): the full D-01 policy table — the 05-02
// permission bridge, the 06-07 mode-aware PAT runtime-headers source over
// the SAME minter the front minted through, the D-06 credential-decrypt
// seam (the loader EXECUTES only when a real cipher ask arrives; the key it
// needs is the USER'S OWN login-keyring material, never ours) — assigned as
// conn.OnRequest BEFORE any traffic.
func wireCodingPlanResponder(conn *zcode.Conn, front *acpfront.Front, machinery *cpMachinery, keys hostsvc.KeySource) {
	responder := interact.NewResponder(
		interact.WithPermissionForwarder(interact.NewPermissionBridge(front.PermissionForward()).Forward),
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

// cpHeaderAskShape records one runtime-headers ANSWER's SHAPE (never its
// values — T-5-17/T-06-24): whether the ask was handled without a wire
// error, and the presence booleans over requestAuth's three fields.
type cpHeaderAskShape struct {
	handled            bool
	wireError          bool
	headersApplied     bool
	hasAPIKey          bool
	hasAPIKeyID        bool
	hasAccountScope    bool
	requestAuthPresent bool
}

// cpAskRecorder is the coding-plan ask tap's state: the interaction/* ask
// log (method + UTC instant) and every runtime-headers answer's shape.
// Guarded by mu: request handlers are concurrent on the conn's read
// goroutine fan-out (conn.go serves one request per fresh goroutine — the
// apicycle askRecorder's guard documents the same discipline, WR-06).
type cpAskRecorder struct {
	mu          sync.Mutex
	lastAsk     time.Time
	headerAsks  []cpHeaderAskShape
	otherAsks   []string
	otherAsksAt []time.Time
}

// recordAsk logs one interaction/* ask structurally.
func (r *cpAskRecorder) recordAsk(method string) {
	now := time.Now().UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastAsk = now

	if method == interact.MethodRequestProviderRuntimeHeaders {
		r.headerAsks = append(r.headerAsks, cpHeaderAskShape{}) // filled by the answer path

		return
	}

	r.otherAsks = append(r.otherAsks, method)
	r.otherAsksAt = append(r.otherAsksAt, now)
}

// recordHeaderAnswer fills the pending runtime-headers shape (the ask/answer
// pair is 1:1 in the conn's request funnel; the LAST pending entry absorbs
// the answer).
func (r *cpAskRecorder) recordHeaderAnswer(res any, werr *zcode.WireError) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.headerAsks) == 0 {
		return
	}

	shape := cpHeaderAskShape{handled: werr == nil, wireError: werr != nil}

	if m, ok := res.(map[string]any); ok {
		shape.headersApplied, _ = m["headersApplied"].(bool)

		if auth, ok := m["requestAuth"].(map[string]any); ok {
			shape.requestAuthPresent = true
			_, shape.hasAPIKey = auth["apiKey"]
			_, shape.hasAPIKeyID = auth["apiKeyId"]
			_, shape.hasAccountScope = auth["accountScope"]
		}
	}

	r.headerAsks[len(r.headerAsks)-1] = shape
}

// lastAskAt returns the UTC instant of the most recent interaction/* ask.
func (r *cpAskRecorder) lastAskAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.lastAsk
}

// headerAskShapes snapshots the recorded runtime-headers answer shapes (a
// copy taken under the lock) for the post-turn assertion — the test
// goroutine reads after Prompt returns while the conn's handler goroutines
// may still be finishing their writes.
func (r *cpAskRecorder) headerAskShapes() []cpHeaderAskShape {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]cpHeaderAskShape(nil), r.headerAsks...)
}

// cpTapInteractionAsks wraps the ALREADY-ASSIGNED conn.OnRequest with the
// ask recorder (the apicycle tap's discipline, extended with the
// runtime-headers answer-shape capture). Call it before any traffic; the
// wrapped chain falls through to the full responder untouched.
func cpTapInteractionAsks(t *testing.T, conn *zcode.Conn) *cpAskRecorder {
	t.Helper()

	next := conn.OnRequest
	if next == nil {
		t.Fatal("cpTapInteractionAsks: wireCodingPlanResponder must assign conn.OnRequest before the tap wraps it")
	}

	tap := &cpAskRecorder{}

	conn.OnRequest = func(reqCtx context.Context, env zcode.Envelope) (any, *zcode.WireError, bool) {
		if strings.HasPrefix(env.Method, "interaction/") {
			now := time.Now().UTC()

			tap.recordAsk(env.Method)

			// Structural logging ONLY (conn.go:436-450 precedent): the
			// cipher asks' params carry the encrypted credential value
			// verbatim — method name + UTC timestamp, never params (T-5-33).
			t.Logf("interaction ask: %s at %s", env.Method, now.Format(time.RFC3339Nano))
		}

		res, werr, handled := next(reqCtx, env)

		if env.Method == interact.MethodRequestProviderRuntimeHeaders {
			if werr != nil {
				tap.recordHeaderAnswer(nil, werr)
			} else {
				tap.recordHeaderAnswer(res, nil)
			}
		}

		return res, werr, handled
	}

	return tap
}

// cpOpenCapture opens the raw capture under THIS plan's V-EXP id (the
// apicycle openCapture's mirror — the header names EXP-CODINGPLAN-CYCLE-01);
// unset env leaves the run uncaptured.
func cpOpenCapture(t *testing.T, inst supervisor.Install) *captureTee {
	t.Helper()

	path := os.Getenv(envRawCapture)
	if path == "" {
		t.Log("no raw capture path set (ZCODE_ACP_RAW_CAPTURE) — the wire is not recorded; " +
			"the ledgered evidence run sets it")

		return nil
	}

	// The apicycle openCapture owns the header discipline; this variant
	// differs ONLY in the capture id, so it delegates through a tiny
	// re-implementation rather than diverging the shared helper.
	return openCaptureWithID(t, inst, path, cpCaptureID)
}

// openCaptureWithID is the id-parameterized capture opener (the
// captureTee/header shape copied from apicycle openCapture — born outside
// the repo, written UNSCRUBBED).
func openCaptureWithID(t *testing.T, inst supervisor.Install, path, id string) *captureTee {
	t.Helper()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open raw capture %s: %v", path, err)
	}

	t.Cleanup(func() { _ = f.Close() })

	header, err := json.Marshal(map[string]string{
		"capture":         id,
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

// cpAuthenticateLeg runs authenticate{zai-individual-coding-plan} and asserts
// the honest read-back verdict: Front.Authenticate returns nil ONLY after
// the registry read-back confirms the account models (the PITFALL-1 chain —
// never the received ack) and the mint-once tail completed.
func cpAuthenticateLeg(t *testing.T, ctx context.Context, front *acpfront.Front) {
	t.Helper()

	if err := front.Authenticate(ctx, acpfront.AuthMethodZaiCodingPlan); err != nil {
		t.Fatalf("authenticate{zai-individual-coding-plan}: %v", err)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeCodingPlan {
		t.Fatalf("auth mode = %q after the coding-plan success, want %q", mode, acpfront.AuthModeCodingPlan)
	}

	t.Log("authenticate ok — the registry read-back verdict proved the account models landed " +
		"(PITFALL 1: the verdict is the registry's) and the PAT mint-once completed")
}

// cpSessionNewLeg opens the session over the scratch workspace and asserts
// the live-registry configOptions (MODEL-01): the model option is MANDATORY
// with the confirmed currentValue and at least one listed entry; the
// thought-level option is RECORDED either way — A4's designed confirmer (its
// presence confirms settings.thoughtLevel live; its absence is the honest
// fallback posture the ledger records, never a silent assumption). Every
// option is logged structurally (ids + currentValues + entry counts).
func cpSessionNewLeg(t *testing.T, ctx context.Context, front *acpfront.Front, ws string) string {
	t.Helper()

	sid, options, err := front.NewSession(ctx, acpport.NewSessionParams{Cwd: ws})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if sid == "" {
		t.Fatal("session/new returned an empty session id")
	}

	var model, thought *acpport.ConfigOptionDef

	for i := range options {
		opt := &options[i]

		t.Logf("configOption: id=%s currentValue=%q entries=%d", opt.ID, opt.CurrentValue, len(opt.Options))

		switch opt.ID {
		case cpOptionIDModel:
			model = opt
		case cpOptionIDThoughtLevel:
			thought = opt
		}
	}

	if model == nil {
		t.Fatalf("session/new returned %d configOptions with NO model entry — MODEL-01 requires the "+
			"live-registry model picker after auth", len(options))
	}

	if model.CurrentValue == "" {
		t.Fatal("the model option carries an empty currentValue — a fabricated or unconfirmed " +
			"selection violates D-10")
	}

	if len(model.Options) == 0 {
		t.Fatal("the model option lists zero entries — the live registry render is empty")
	}

	if thought != nil {
		t.Logf("A4 CONFIRMED-SHAPE: thought_level option present (currentValue=%q, entries=%d) — "+
			"settings.thoughtLevel rendered live", thought.CurrentValue, len(thought.Options))
	} else {
		t.Log("A4 ADJUST-SHAPE: no thought_level option on the live render — the settings.thoughtLevel " +
			"block is absent in this reply and the honest omission posture stands (record in the ledger; " +
			"never silently assumed)")
	}

	t.Logf("session/new ok: %s (create + the D-03 subscribe gate; model=%q with %d entries)",
		sid, model.CurrentValue, len(model.Options))

	return sid
}

// cpToollessTurnLeg runs the cycle's ONLY turn (the D-13 cap) and asserts
// the honest outcome tier: end_turn, ZERO tool_call frames (tool-less by
// definition), and the runtime-headers ask answered — at least one ask
// fired mid-turn and EVERY ask was handled with the three-field requestAuth
// shape (apiKey + apiKeyId + accountScope present — values never recorded).
func cpToollessTurnLeg(t *testing.T, ctx context.Context, front *acpfront.Front, asks *cpAskRecorder, sid string) {
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

	shapes := asks.headerAskShapes()

	if len(shapes) == 0 {
		t.Fatal("no interaction/requestProviderRuntimeHeaders ask fired on the coding-plan turn — " +
			"the PAT-serving ask is the D-12 cycle's live half; record the honest verdict")
	}

	for i, shape := range shapes {
		if shape.wireError || !shape.handled {
			t.Fatalf("runtime-headers ask %d/%d answered a wire error or went unhandled — "+
				"the PAT source must serve the cached mint", i+1, len(shapes))
		}

		if !shape.headersApplied || !shape.requestAuthPresent ||
			!shape.hasAPIKey || !shape.hasAPIKeyID || !shape.hasAccountScope {
			t.Fatalf("runtime-headers ask %d/%d answered an incomplete shape "+
				"(headersApplied=%v requestAuth=%v apiKey=%v apiKeyId=%v accountScope=%v) — "+
				"the three-field requestAuth is the coding-plan answer's contract",
				i+1, len(shapes), shape.headersApplied, shape.requestAuthPresent,
				shape.hasAPIKey, shape.hasAPIKeyID, shape.hasAccountScope)
		}
	}

	t.Logf("tiny tool-less turn ok: end_turn, zero tool frames, %d runtime-headers ask(s) all "+
		"answered with the three-field requestAuth shape (token material never recorded)",
		len(shapes))
}

// cpDesktopVersionPattern is main.go desktopVersionPattern's mirror (the
// version-looking segment inside the layout names).
var cpDesktopVersionPattern = regexp.MustCompile(`\d+\.\d+(\.\d+)?`)

// cpDesktopVersionFallback is the reference's documented fallback (ref
// installedAppVersion's "0.0.0" — A6: no server-side validation is
// documented; the real run confirms).
const cpDesktopVersionFallback = "0.0.0"

// cpClientVersion resolves the mint chain's clientVersion seam input (the
// DESKTOP's version per main.go patClientVersion's posture): version-first
// like main.go's desktopVersion — Install.Version, then the packed AppImage
// basename, then the best-effort install dig — with the documented fallback
// otherwise. Lazy by contract — the minter calls it only when a mint actually
// runs.
func cpClientVersion(inst supervisor.Install) func() string {
	return func() string {
		if inst.Version != "" {
			return inst.Version
		}

		if v := cpVersionFromPath(inst.AppImage); v != "" {
			return v
		}

		for _, path := range []string{inst.Binary, inst.Root, inst.Entrypoint} {
			if v := cpVersionFromPath(path); v != "" {
				return v
			}
		}

		return cpDesktopVersionFallback
	}
}

// cpVersionFromPath digs a version-looking segment out of one install path,
// deepest segment first (main.go versionFromPath's mirror).
func cpVersionFromPath(path string) string {
	segs := strings.Split(path, string(filepath.Separator))

	for i := len(segs) - 1; i >= 0; i-- {
		if v := cpDesktopVersionPattern.FindString(segs[i]); v != "" {
			return v
		}
	}

	return ""
}
