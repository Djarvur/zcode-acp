//go:build !windows

// Spawn-contract tests (BRIDGE-03 part 2, plan 03-07 Task 1): the canonical
// argv line, the allowlisted child env with the proxy scrub, and the
// process-group start — all asserted on a REAL child's OBSERVED state through
// the exported SpawnChild path (T-3-08). The spawn contract is the Phase-1
// verified canonical line in docs/protocol/install-discovery.md :48-52; the
// env allowlist and scrub are D-03 / PITFALL 9 (Go caches proxy settings once
// — a hostile parent env must never reach the child).
//
// The helper child is the GOROOT-verified reexec idiom (03-06 precedent): the
// test binary re-executed through SpawnChild with the contract env itself as
// the trigger, so the environment the child reports is EXACTLY the builder's
// output — no test-only vars ride along. (A suite run under a parent that
// already exports ELECTRON_RUN_AS_NODE=1 is pathological and unsupported, the
// same trade the stdlib helper idiom accepts for its own trigger var.)
package supervisor_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Spawn-contract facts under test (install-discovery.md :48-52, D-03).
const (
	// childRunAsNodeEnv is the Electron-as-Node switch the child MUST observe.
	childRunAsNodeEnv = "ELECTRON_RUN_AS_NODE"
	// childBuiltinEnv carries the newest zcode-builtin.json into the child.
	childBuiltinEnv = "ZCODE_BUILTIN_PROVIDER_CONFIG_FILE"
	// childPersonalEnv carries the personal provider config into the child —
	// resolveNodeProviderRuntimePaths requires BOTH paths paired; a worker
	// started with only the builtin path runs degraded (account models never
	// publish — EXP-CODINGPLAN-CYCLE-01 attempts 1-6 field proof).
	childPersonalEnv = "ZCODE_PERSONAL_PROVIDER_CONFIG_FILE"
	// helperRunFlag targets the helper test inside the re-executed binary.
	helperRunFlag = "-test.run=^TestSpawnEnvHelper$"

	// fixturePathValue and fixtureHomeValue are the allowlist sources the
	// hostile parent env provides; the child must observe them verbatim.
	fixturePathValue = "/fixture-bin:/usr/bin:/bin"
	fixtureHomeValue = "/home/fixture"

	// ignoredParentVar rides in the hostile input and must never cross.
	ignoredParentVar = "IGNORED_PARENT_VAR=must-not-cross"

	// proxyVarsInput is every proxy variable the hostile parent env exports
	// (both cases on Linux — the child's getenv is case-sensitive); the
	// builder must scrub all of them (PITFALL 9 / D-03 / T-3-08).
	proxyVarsInput = "HTTP_PROXY HTTPS_PROXY ALL_PROXY http_proxy https_proxy all_proxy"

	// spawnExitDeadline bounds the Task-1 teardown's wait for the helper's
	// exit; the full staged shutdown is Task 2's subject.
	spawnExitDeadline = 5 * time.Second

	// fixtureLayout is the desktop Install layout of the fixture shape.
	fixtureLayout = "deb"
)

// errNoReportLine is readSpawnReport's EOF failure sentinel.
var errNoReportLine = errors.New("child stdout reached EOF without a report line")

// fixtureInstall is a desktop Install shape (the deb layout's /opt tree with
// the invariant entrypoint and an injected builtin path). Its paths are never
// touched — SpawnArgv/ChildEnv consume them as data; the spawn target is the
// re-executed test binary (the real Electron bundle never runs in CI,
// Pitfall 10).
func fixtureInstall() supervisor.Install {
	return supervisor.Install{
		Root:       "/opt/ZCode-fixture",
		Binary:     "/opt/ZCode-fixture/zcode",
		Entrypoint: "/opt/ZCode-fixture/resources/glm/zcode.cjs",
		BuiltinFile: "/home/fixture/.zcode/v2/runtime/provider/linux-x86_64" +
			"/3.18.2/endpoint-78d7c3bef4024722642626fe3669a799/zcode-builtin.json",
		Layout: fixtureLayout,
	}
}

// hostileParentEnv is the builder input: the allowlist sources PLUS every
// proxy variable plus one var that must never cross. Constructed from
// scratch (not os.Environ) so the child's observed env is exactly the
// builder's output and every assertion is deterministic.
func hostileParentEnv() []string {
	return []string{
		"PATH=" + fixturePathValue,
		"HOME=" + fixtureHomeValue,
		"HTTP_PROXY=http://192.0.2.1:9",
		"HTTPS_PROXY=http://192.0.2.1:9",
		"ALL_PROXY=socks5://192.0.2.1:9",
		"http_proxy=http://192.0.2.1:9",
		"https_proxy=http://192.0.2.1:9",
		"all_proxy=socks5://192.0.2.1:9",
		ignoredParentVar,
	}
}

// allowedChildKeys is the exact key set the child may observe: the allowlist
// copies (PATH, HOME) plus the two ZCode contract vars — nothing else.
func allowedChildKeys() []string {
	return []string{"PATH", "HOME", childRunAsNodeEnv, childBuiltinEnv, childPersonalEnv}
}

// spawnReport is the helper child's JSON self-report: the environment it
// OBSERVES plus its pid and process-group id.
type spawnReport struct {
	Env  map[string]string `json:"env"`
	PID  int               `json:"pid"`
	PgID int               `json:"pgid"`
}

// runSpawnEnvHelper is the re-executed child branch: keyed on
// ELECTRON_RUN_AS_NODE itself (the contract env is the trigger, so the env
// passed to SpawnChild is ChildEnv's output VERBATIM), it reports the
// environment it observes plus pid/pgid as one JSON line on stdout, then
// exits the helper process.
func runSpawnEnvHelper() {
	env := make(map[string]string)

	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			env[k] = v
		}
	}

	report := spawnReport{Env: env, PID: os.Getpid(), PgID: syscall.Getpgrp()}

	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

// TestSpawnEnvHelper is the -test.run target the parent re-executes as the
// child (helperRunFlag selects only this test inside the helper process).
func TestSpawnEnvHelper(t *testing.T) {
	if os.Getenv(childRunAsNodeEnv) == "1" {
		runSpawnEnvHelper()
	}
}

// TestSpawnEnv proves the allowlist + scrub on the child's OBSERVED
// environment (never on the builder's intention): ELECTRON_RUN_AS_NODE=1 and
// the builtin var set from the injected path, PATH and HOME copied, every
// proxy variable scrubbed even though the parent env input exports them, no
// other parent var crossing, and the process-group start verified on the
// real child (pgid == pid — Setpgid made it its own group leader).
func TestSpawnEnv(t *testing.T) {
	if os.Getenv(childRunAsNodeEnv) == "1" {
		runSpawnEnvHelper() // defensive: the -test.run filter never selects this
	}

	inst := fixtureInstall()
	childEnv := supervisor.ChildEnv(inst, hostileParentEnv())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	child, err := supervisor.SpawnChild(ctx, []string{os.Args[0], helperRunFlag}, childEnv, supervisor.SpawnOpts{})
	if err != nil {
		t.Fatalf("spawn the helper child through the exported SpawnChild path: %v", err)
	}

	t.Cleanup(func() { terminateSpawnChild(t, child) })

	report := readSpawnReport(t, child)

	assertObservedEnv(t, inst, report)

	if report.PgID != report.PID {
		t.Fatalf("child pgid = %d, pid = %d — Setpgid must make the child its own group leader",
			report.PgID, report.PID)
	}
}

// assertObservedEnv pins the child's OBSERVED environment (never the
// builder's intention): the two contract vars, the verbatim allowlist copies,
// every proxy variable scrubbed, and nothing outside the allowlist.
func assertObservedEnv(t *testing.T, inst supervisor.Install, report spawnReport) {
	t.Helper()

	if got := report.Env[childRunAsNodeEnv]; got != "1" {
		t.Fatalf("child observed %s = %q, want %q (install-discovery.md :48-52)", childRunAsNodeEnv, got, "1")
	}

	if got := report.Env[childBuiltinEnv]; got != inst.BuiltinFile {
		t.Fatalf("child observed %s = %q, want the injected builtin path %q", childBuiltinEnv, got, inst.BuiltinFile)
	}

	wantPersonal := filepath.Join(report.Env["HOME"], ".zcode", "v2", "provider_config.json")
	if got := report.Env[childPersonalEnv]; got != wantPersonal {
		// resolveNodeProviderRuntimePaths requires the builtin+personal pair.
		t.Fatalf("child observed %s = %q, want the desktop-parity personal path %q",
			childPersonalEnv, got, wantPersonal)
	}

	if got := report.Env["PATH"]; got != fixturePathValue {
		t.Fatalf("child observed PATH = %q, want the parent's value copied verbatim", got)
	}

	if got := report.Env["HOME"]; got != fixtureHomeValue {
		t.Fatalf("child observed HOME = %q, want the parent's value copied verbatim", got)
	}

	for _, key := range strings.Fields(proxyVarsInput) {
		if got, present := report.Env[key]; present {
			t.Fatalf("child observed %s=%q — the proxy scrub must remove it (PITFALL 9)", key, got)
		}
	}

	if _, present := report.Env["IGNORED_PARENT_VAR"]; present {
		t.Fatal("child observed a var outside the allowlist — the env contract is an allowlist, not a scrub filter")
	}

	observedKeys := make([]string, 0, len(report.Env))

	for key := range report.Env {
		observedKeys = append(observedKeys, key)
	}

	slices.Sort(observedKeys)
	wantKeys := allowedChildKeys()
	slices.Sort(wantKeys)

	if !slices.Equal(observedKeys, wantKeys) {
		t.Fatalf("child observed env keys %v, want exactly the allowlist %v", observedKeys, wantKeys)
	}
}

// TestSpawnArgvShape pins the canonical spawn line's argv: exactly
// [Binary, Entrypoint, "app-server", "--stdio"] for a desktop Install
// (install-discovery.md :48-52 — the contract Phase 4's front leg inherits).
func TestSpawnArgvShape(t *testing.T) {
	inst := fixtureInstall()

	got := supervisor.SpawnArgv(inst)
	want := []string{inst.Binary, inst.Entrypoint, "app-server", "--stdio"}

	if !slices.Equal(got, want) {
		t.Fatalf("SpawnArgv(inst) = %v, want exactly %v (install-discovery.md :48-52)", got, want)
	}
}

// readSpawnReport reads the helper's single JSON self-report line from the
// child's stdout (the Conn's future read side), failing fast on a deadline.
func readSpawnReport(t *testing.T, child *supervisor.Child) spawnReport {
	t.Helper()

	type result struct {
		report spawnReport
		err    error
	}

	done := make(chan result, 1)

	go func() {
		var report spawnReport

		scanner := bufio.NewScanner(child.Stdout)
		if scanner.Scan() {
			reportErr := json.Unmarshal(scanner.Bytes(), &report)
			done <- result{report: report, err: reportErr}

			return
		}

		done <- result{err: errNoReportLine}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("read the helper's self-report: %v", res.err)
		}

		return res.report

	case <-time.After(spawnExitDeadline):
		t.Fatalf("no self-report from the helper within %v", spawnExitDeadline)
	}

	return spawnReport{} // unreachable: both select arms end the test
}

// terminateSpawnChild is Task 1's minimal teardown: stdin end, wait for the
// exit the monitor reports, SIGKILL backstop on the whole group. The full
// staged shutdown (grace window, ESRCH-verified zero survivors) is Task 2.
func terminateSpawnChild(t *testing.T, child *supervisor.Child) {
	t.Helper()

	_ = child.Stdin.Close()

	select {
	case <-child.Exited:
	case <-time.After(spawnExitDeadline):
		_ = syscall.Kill(-child.PgID, syscall.SIGKILL)
		<-child.Exited
	}
}
