//go:build realdesktop

// The guarded zero-spend read-back DIAGNOSTIC (Phase 5 plan 05-11) — the
// field instrument for the SC4 auth read-back gap (REAL RUN attempt 1,
// EXP-APIKEY-CYCLE-01: the push acked received but no registry surface
// confirmed within the window). MANUAL-ONLY, never CI (the 03-07
// discipline, apicycle_test.go sibling): the realdesktop build tag excludes
// this file from default builds, vet, and CI; the ZCODE_ACP_READBACK_DIAG=1
// runtime guard skips the body even when someone builds with the tag (the
// apicycle double-guard discipline), skipping honestly in ~2 ms.
//
// ZERO-SPEND CONTRACT (structural): this file contains NO turn-sending
// wire call — not in code, not in a comment. The diagnostic's cycle is
// REGISTRY-LEVEL ONLY by construction: push → refresh → observe →
// probe-create → close. The probe session is created, observed, and closed;
// no turn is ever started (registry-level ops spend nothing — the Phase-1
// precedent). The verify gate greps for the absent wire method name; the
// absence IS the contract.
//
// Canonical run command (executor-run for plan 05-11's Task 2 evidence;
// the api key is OPTIONAL — an unset ZCODE_ACP_API_KEY is fine, the honest
// no-key coded answer on a runtime-headers ask is itself diagnostic
// evidence; raw captures are born OUTSIDE the repo):
//
//	export ZCODE_ACP_RAW_CAPTURE=<scratch path OUTSIDE this repo>  # raw born outside
//	ZCODE_ACP_READBACK_DIAG=1 go test -tags realdesktop -count=1 -run TestReadbackDiagnosis ./test/realdesktop/
//
// Evidence discipline: the wire tee records every back-leg byte; the raw
// capture passes experiments/scrub-capture.mjs BEFORE any commit and lands
// as experiments/captures/EXP-READBACK-DIAG-01.jsonl (scan-secrets green).
// The ledger line in experiments/README.md records the zero spend, the
// capture id, and the branch verdict — never credentials.
//
// Structured logging only: notification METHOD NAMES, ack fields, and model
// refs — never params, never key material (the apicycle capture discipline).

package realdesktop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Manual-tier knobs for the diagnostic (the apicycle discipline: env
// double-guard, optional capture env — reused — and the window scale).
const (
	// diagEnv is the runtime guard: the body never runs without it.
	diagEnv = "ZCODE_ACP_READBACK_DIAG"

	// diagCaptureID is the V-EXP id the evidence pipeline records for this
	// instrument (the ledger entry names it; the scrubbed capture lands
	// under experiments/captures/ under the same id).
	diagCaptureID = "EXP-READBACK-DIAG-01"

	// diagDeadline bounds one diagnostic pass (discover → spawn → push →
	// refresh → the observation window → probe-create → verdict → staged
	// shutdown). Electron startup takes seconds; the window alone is 30 s.
	diagDeadline = 6 * time.Minute

	// diagWindow is the registry-surface observation window (the
	// hostsvc.DefaultReadbackDeadline shape — the same window the REAL RUN
	// attempt 1 waited, so the evidence is comparable).
	diagWindow = 30 * time.Second

	// diagRuntimeTree is the desktop's runtime-provider tree under the user
	// home (the version dirs the desktop's own runtime resolves its ACTIVE
	// builtin from — candidate-cause (a)'s terrain). Read-only fact-finding
	// for the verdict block: structural paths and revision ints, never
	// contents.
	diagRuntimeTree = ".zcode/runtime/releases"
)

// TestReadbackDiagnosis runs the guarded zero-spend diagnostic: the full
// discovery → supervised spawn → conn chain (the same stack the REAL RUN
// builds, reusing this package's helpers verbatim), then the REGISTRY-LEVEL
// observation sequence — WatchRegistry registered BEFORE any traffic (the
// conn hook contract), the strict-envelope push, the refresh, a 30 s
// notification window logging every envelope's method name, the probe
// create (temp-dir cwd, no model binding, no turn), and the probe close.
// The product is the structured DIAG verdict block: the facts that decide
// WHICH candidate cause the attempt-1 gap's evidence supports — (a) the
// basedOn hash mismatching the desktop's active builtin, (b) the
// empty-providers overlay being nothing-to-apply, (c) the read-back surface
// being the first create reply — or none.
//
// The test FAILS only on setup/transport failure (discovery, spawn, conn);
// every observation outcome is REPORTED via the DIAG block, never asserted
// — the instrument's product is the verdict, not a pass/fail on desktop
// behavior.
func TestReadbackDiagnosis(t *testing.T) {
	if os.Getenv(diagEnv) != "1" {
		t.Skip("manual-only diagnostic: set ZCODE_ACP_READBACK_DIAG=1 AND build with -tags realdesktop — CI never contacts the desktop")
	}

	ctx, cancel := context.WithTimeout(context.Background(), diagDeadline)
	defer cancel()

	sel := resolveModelSelection()
	want := []string{sel.ProviderID + "/" + sel.ModelID}

	inst := discoverRealInstall(ctx, t)
	logObservedInstall(t, inst)

	// Host-level builtin candidates (candidate (a)'s terrain): the desktop
	// resolves its ACTIVE builtin from its runtime tree, which may hold
	// different revisions/paths than the install file our discovery feeds
	// the snapshot formula. Recorded as structural facts — never assumed.
	runtimeBuiltins := diagRuntimeBuiltins(t)
	pushedFormula := diagPushedFormula(t, inst)

	capture := openDiagCapture(t, inst)

	// Two lifecycles, deliberately (the production run() discipline the
	// apicycle harness pins): the supervision ctx drives the staged
	// teardown, the spawn ctx is the exec-level backstop.
	supCtx, supCancel := context.WithCancel(context.Background())
	spawnCtx, spawnCancel := context.WithCancel(context.Background())
	defer spawnCancel()

	child := startSupervisedChild(t, supCtx, spawnCtx, inst)

	//nolint:contextcheck // NewConn owns its base context (ctx only bounds the legs below)
	conn := zcode.NewConn(newTeeDuplex(capture, child))
	defer func() { _ = conn.Close() }() // the staged shutdown owns stdin from here

	// One key, one source (AUTH-01): the env-sourced binding; UNSET IS FINE
	// — no leg here asks for runtime headers (no turn is ever started), and
	// a coded no-key answer would itself be diagnostic evidence.
	keys := hostsvc.NewStaticKey(os.Getenv(envAPIKey), "")

	front := buildCycleFront(conn, inst, keys, sel)

	// The full responder BEFORE any traffic — the probe create's host
	// obligations (the runtime-preferences ask) must be answered or the
	// create fails -32603 (the Phase-1 wire fact).
	wireCycleResponder(conn, front, keys)

	// The watcher registers BEFORE any traffic (the conn hook contract —
	// the same registration-first shape production authenticate runs).
	watcher, remove := hostsvc.WatchRegistry(conn, want)
	defer remove()

	// The observation observer: every notification envelope's METHOD NAME
	// only (never params), flagging state.updated patches that carry
	// provider models. Removed before the conn closes (LIFO below) so no
	// delivery outlives the test.
	obs := &diagObserver{t: t}
	logRemove := conn.AddObserver(obs.observe)
	defer logRemove()

	// Push the strict envelope production builds (structural facts only).
	snap, err := hostsvc.NewSnapshotBuilder(keys).Build(inst)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}

	t.Logf("DIAG push: revision=%s basedOn=%s providers-empty=%t states-empty=%t",
		snap.Revision, snap.BasedOn, diagRecordEmpty(t, snap.Params, "providers"), diagRecordEmpty(t, snap.Params, "states"))

	ack, err := hostsvc.PushEnvelope(ctx, conn, snap.Params)
	if err != nil {
		t.Fatalf("push snapshot: %v", err)
	}

	t.Logf("DIAG ack: status=%q providerCount=%d receivedRevision-echo=%t",
		ack.Status, ack.ProviderCount, ack.ReceivedRevision == snap.Revision)

	if err := hostsvc.RefreshRegistry(ctx, conn); err != nil {
		t.Fatalf("refresh registry: %v", err)
	}

	t.Logf("DIAG refresh: ok")

	// The observation window: every notification method name lands in the
	// DIAG log; the watcher latches underneath if any surface carries want.
	diagSleepCtx(t, ctx, diagWindow)

	t.Logf("DIAG state-updated-in-window: %s + provider-patch: %s", yesNo(obs.stateUpdated()), yesNo(obs.providerPatch()))

	// The probe create (the dialect §8 read-back surface): a raw conn call
	// with a temp-dir cwd — NO model binding, NO turn. Its reply is the
	// field evidence for candidate (c); transport refusal is evidence too,
	// reported, never fatal.
	probeRes, probeErr := conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{"workspacePath": t.TempDir(), "workspaceKey": t.TempDir()},
	})

	probeID := ""
	createModels := []string(nil)

	switch {
	case probeErr != nil:
		t.Logf("DIAG probe-create: refused (%v)", probeErr)
	default:
		createModels = diagCreateModels(t, probeRes)
		probeID = diagSessionID(t, probeRes)

		t.Logf("DIAG create-available: %s", diagModelList(createModels))

		// The reply feeds the watcher explicitly (the OfferCreateReply
		// surface — the verdict stays the registry's, never the transport
		// success).
		watcher.OfferCreateReply(probeRes)

		if probeID != "" {
			// The dialect's camelCase session-id params key (the wire
			// spelling front.go's wireKeySessionID twin pins).
			if _, closeErr := conn.Call(ctx, zcode.MethodSessionClose, map[string]any{
				"sessionId": probeID,
			}); closeErr != nil {
				t.Logf("DIAG probe-close: failed (%v)", closeErr)
			} else {
				t.Logf("DIAG probe-close: ok (probe session torn down)")
			}
		}
	}

	// The watcher's verdict (the PITFALL-1 discipline: never the ack, never
	// the probe's transport success).
	rb, verdictErr := watcher.Await(ctx)
	if verdictErr != nil || !rb.Applied {
		t.Logf("DIAG watcher-verdict: not-confirmed (%v)", verdictErr)
	} else {
		t.Logf("DIAG watcher-verdict: confirmed (revision echo %q)", rb.Revision)
	}

	// The branch verdict: which candidate cause the collected facts support.
	t.Logf("DIAG branch-evidence: %s", diagBranchVerdict(diagFacts{
		want:            want,
		providerPatch:   obs.providerPatch(),
		confirmed:       verdictErr == nil && rb.Applied,
		createModels:    createModels,
		pushedBasedOn:   snap.BasedOn,
		runtimeBuiltins: runtimeBuiltins,
		pushedFormula:   pushedFormula,
	}))

	// Staged shutdown via the supervisor, then the zero-orphan proof.
	supCancel()
	waitStagedShutdown(t, child)
	spawnCancel() // the teardown backstop: the group is long gone; ESRCH-quiet

	assertGroupGone(t, child)

	capture.report(t)

	t.Log("diagnostic complete — the DIAG block above is the product (registry-level only, zero spend by construction)")
}

// diagObserver collects the observation-window facts: which notification
// methods arrived (names only, logged as they land) and whether any
// state.updated patch carried provider models (a structural presence check,
// never param contents).
type diagObserver struct {
	t                *testing.T
	stateUpdatedSeen int
	providerPatchN   int
}

// observe runs on the read goroutine — method-name logging plus the two
// structural facts; nothing blocks.
func (o *diagObserver) observe(env zcode.Envelope) {
	if env.Method == "" {
		return
	}

	o.t.Logf("DIAG notif: %s", env.Method)

	if env.Method != zcode.MethodStateUpdated {
		return
	}

	o.stateUpdatedSeen++

	if diagPatchProviderRefs(env.Params) > 0 {
		o.providerPatchN++
	}
}

// stateUpdated reports whether any state.updated arrived in the window.
func (o *diagObserver) stateUpdated() bool {
	return o.stateUpdatedSeen > 0
}

// providerPatch reports whether a state.updated carried provider models.
func (o *diagObserver) providerPatch() bool {
	return o.providerPatchN > 0
}

// diagPatchProviderRefs digs the patch.providers.<id>.models ref count out
// of a state.updated params payload (presence only — the count, never the
// contents).
func diagPatchProviderRefs(raw json.RawMessage) int {
	var params struct {
		Patch struct {
			Providers map[string]json.RawMessage `json:"providers"`
		} `json:"patch"`
	}

	if err := json.Unmarshal(raw, &params); err != nil {
		return 0
	}

	return len(params.Patch.Providers)
}

// openDiagCapture opens the diagnostic's raw capture when the env path is
// set, writing this instrument's own header (capture id EXP-READBACK-DIAG-01
// — the apicycle openCapture helper pins the full-cycle id, so this
// instrument writes its header over the SAME captureTee type and record
// discipline rather than mislabeling the evidence). Unset leaves the run
// uncaptured.
func openDiagCapture(t *testing.T, inst supervisor.Install) *captureTee {
	t.Helper()

	path := os.Getenv(envRawCapture)
	if path == "" {
		t.Log("no raw capture path set (ZCODE_ACP_RAW_CAPTURE) — the wire is not recorded; the evidence run sets it")

		return nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("open raw capture %s: %v", path, err)
	}

	t.Cleanup(func() { _ = f.Close() })

	header, err := json.Marshal(map[string]string{
		"capture":         diagCaptureID,
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

// diagSleepCtx waits d under ctx — the observation window (a ctx expiry is
// a harness-budget failure, the setup/transport class the test fails on).
func diagSleepCtx(t *testing.T, ctx context.Context, d time.Duration) {
	t.Helper()

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
		t.Fatalf("diagnostic window interrupted by the harness deadline: %v", ctx.Err())
	}
}

// diagRuntimeBuiltins enumerates the desktop runtime tree's builtin
// candidates (path + revision int + computed formula): candidate (a)'s
// terrain — the ACTIVE builtin the desktop's own runtime resolves may
// differ from the install file our discovery feeds the formula. Absent
// trees record honestly as empty.
func diagRuntimeBuiltins(t *testing.T) []diagBuiltin {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Logf("DIAG runtime-tree: unresolvable home (%v)", err)

		return nil
	}

	matches, err := filepath.Glob(filepath.Join(home, diagRuntimeTree, "*", "agent", "provider", "zcode-builtin.json"))
	if err != nil {
		t.Logf("DIAG runtime-tree: glob failed (%v)", err)

		return nil
	}

	if len(matches) == 0 {
		t.Logf("DIAG runtime-tree: no builtin candidates under the runtime releases tree")

		return nil
	}

	out := make([]diagBuiltin, 0, len(matches))
	for _, path := range matches {
		out = append(out, diagBuiltinAt(t, path))
	}

	return out
}

// diagBuiltin is one builtin candidate's structural facts.
type diagBuiltin struct {
	Path     string
	Revision int
	Formula  string
}

// diagBuiltinAt reads one candidate's revision int and computes its
// formula; an unreadable candidate records honestly (revision 0, empty
// formula).
func diagBuiltinAt(t *testing.T, path string) diagBuiltin {
	t.Helper()

	out := diagBuiltin{Path: path}

	data, err := os.ReadFile(path) //nolint:gosec // the path comes from the glob over the desktop's own runtime tree
	if err != nil {
		t.Logf("DIAG runtime-tree: %s unreadable (%v)", path, err)

		return out
	}

	var parsed struct {
		Revision int `json:"revision"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Logf("DIAG runtime-tree: %s unparseable (%v)", path, err)

		return out
	}

	out.Revision = parsed.Revision
	out.Formula = supervisor.BuiltinRevision(path, parsed.Revision)

	return out
}

// diagPushedFormula computes the formula the push will carry (over the
// install's builtin file — the same input production Build reads) without
// building the envelope twice; recorded so the DIAG block can compare it
// against the runtime-tree candidates.
func diagPushedFormula(t *testing.T, inst supervisor.Install) string {
	t.Helper()

	rev, err := readBuiltinRevision(inst.BuiltinFile)
	if err != nil {
		t.Logf("DIAG install-builtin: unreadable (%v)", err)

		return ""
	}

	formula := supervisor.BuiltinRevision(inst.BuiltinFile, rev)
	t.Logf("DIAG install-builtin: revision=%d formula=%s", rev, formula)

	return formula
}

// diagFacts is the verdict block's input: everything the observation
// sequence collected.
type diagFacts struct {
	want            []string
	providerPatch   bool
	confirmed       bool
	createModels    []string
	pushedBasedOn   string
	runtimeBuiltins []diagBuiltin
	pushedFormula   string
}

// diagBranchVerdict derives the branch evidence from the logged facts,
// carrying its reason in the verdict string:
//
//   - "c-create-reply-surface": the registry serves the want models on the
//     create reply (settings.model.available) and NO provider patch fired —
//     the dialect §8 surface is the read-back; the patch surface does not
//     fire for this push on this desktop.
//   - "none": a provider patch carried the want models — the scripted
//     patch surface exists here; attempt 1's miss was environmental.
//   - "a-hash-mismatch": neither surface carries the models AND the runtime
//     tree holds builtin candidates whose revision/path differ from the
//     pushed basedOn — the silent-discard shape dialect §8 records.
//   - "b-empty-providers": neither surface carries the models, the pushed
//     overlay was empty, and no candidate mismatch stands out —
//     nothing-to-apply is the leaning cause.
func diagBranchVerdict(f diagFacts) string {
	switch {
	case f.providerPatch && f.confirmed:
		return "none — a state.updated provider patch carried the want models; the patch surface exists on this desktop and confirmed the verdict"
	case !f.providerPatch && f.confirmed && containsWant(f.createModels, f.want):
		return "c-create-reply-surface — the create reply carried the want models and no provider patch fired; the create surface is the read-back"
	case !f.confirmed && diagCandidateMismatch(f):
		return "a-hash-mismatch — neither surface carried the want models and the runtime tree holds builtin candidates whose revision/path differ from the pushed basedOn (the dialect §8 silent-discard shape)"
	case !f.confirmed:
		return "b-empty-providers — neither surface carried the want models and the pushed providers overlay was empty (nothing-to-apply is the leaning cause); no candidate mismatch stands out"
	default:
		return "none — evidence ambiguous (confirmed without a provider patch or matching create models); route the open question to the human checkpoint"
	}
}

// diagCandidateMismatch reports whether any runtime-tree candidate's
// formula differs from the pushed basedOn (the candidate-(a) discriminator).
func diagCandidateMismatch(f diagFacts) bool {
	for _, b := range f.runtimeBuiltins {
		if b.Formula != "" && f.pushedBasedOn != "" && b.Formula != f.pushedBasedOn {
			return true
		}
	}

	return false
}

// containsWant reports whether the observed create refs carry every want
// model.
func containsWant(got, want []string) bool {
	set := make(map[string]struct{}, len(got))
	for _, g := range got {
		set[g] = struct{}{}
	}

	for _, m := range want {
		if _, ok := set[m]; !ok {
			return false
		}
	}

	return len(want) > 0
}

// diagCreateModels digs the create reply's settings.model.available refs
// (structural model refs only — the capture-verbatim camelCase keys).
func diagCreateModels(t *testing.T, res json.RawMessage) []string {
	t.Helper()

	var obj map[string]any
	if err := json.Unmarshal(res, &obj); err != nil {
		t.Logf("DIAG create-reply: undecodable (%v)", err)

		return nil
	}

	settings, _ := obj["settings"].(map[string]any)

	model, _ := settings["model"].(map[string]any)

	available, _ := model["available"].([]any)

	out := make([]string, 0, len(available))
	for _, entry := range available {
		avail, _ := entry.(map[string]any)

		ref, _ := avail["ref"].(map[string]any)

		pid, _ := ref["providerId"].(string)
		mid, _ := ref["modelId"].(string)

		if pid == "" && mid == "" {
			continue
		}

		out = append(out, pid+"/"+mid)
	}

	return out
}

// diagSessionID digs result.session.sessionId out of the probe create
// reply (the front.go createSessionID dig-out, mirrored locally READ-ONLY
// — front.go is a same-wave sibling's file and stays untouched).
func diagSessionID(t *testing.T, res json.RawMessage) string {
	t.Helper()

	var result struct {
		Session json.RawMessage `json:"session"`
	}

	if err := json.Unmarshal(res, &result); err != nil {
		t.Logf("DIAG create-reply: no session object (%v)", err)

		return ""
	}

	var session map[string]any
	if err := json.Unmarshal(result.Session, &session); err != nil {
		t.Logf("DIAG create-reply: session undecodable (%v)", err)

		return ""
	}

	sid, _ := session["sessionId"].(string)

	return sid
}

// diagRecordEmpty reports whether one envelope member marshals as the empty
// record (the structural emptiness fact for the DIAG push line).
func diagRecordEmpty(t *testing.T, params map[string]any, key string) bool {
	t.Helper()

	raw, err := json.Marshal(params[key])
	if err != nil {
		return false
	}

	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		return false
	}

	return len(rec) == 0
}

// diagModelList renders observed model refs for the DIAG line (or "none").
func diagModelList(models []string) string {
	if len(models) == 0 {
		return "none"
	}

	return strings.Join(models, ",")
}

// yesNo renders a boolean fact for the DIAG lines.
func yesNo(b bool) string {
	if b {
		return "yes"
	}

	return "no"
}
