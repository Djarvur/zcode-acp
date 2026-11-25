//go:build windows

package supervisor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

const (
	windowsTreeHelperEnv           = "ZCODE_ACP_WINDOWS_TREE_HELPER"
	windowsTreeGrandchildHelperEnv = "ZCODE_ACP_WINDOWS_TREE_GRANDCHILD"
)

func TestWindowsExplicitRootInstall(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "ZCode.exe")
	entrypoint := filepath.Join(root, filepath.FromSlash(entrypointRel))
	if err := os.WriteFile(binary, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(entrypoint), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entrypoint, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}

	inst, ok := ExplicitRootInstall(root)
	if !ok {
		t.Fatal("ExplicitRootInstall rejected a Windows ZCode tree")
	}
	if inst.Binary != binary {
		t.Fatalf("binary = %q, want %q", inst.Binary, binary)
	}
	if inst.Entrypoint != entrypoint {
		t.Fatalf("entrypoint = %q, want %q", inst.Entrypoint, entrypoint)
	}

	override, ok := OverrideInstall(binary)
	if !ok || override.Binary != binary || override.Root != root {
		t.Fatalf("OverrideInstall(%q) = %#v, %v", binary, override, ok)
	}
}

func TestWindowsChildEnv(t *testing.T) {
	inst := Install{BuiltinFile: `C:\ZCode\resources\config\provider\zcode-builtin.json`}
	parent := []string{
		`Path=C:\Windows\System32`,
		`USERPROFILE=C:\Users\fixture`,
		`APPDATA=C:\Users\fixture\AppData\Roaming`,
		`LOCALAPPDATA=C:\Users\fixture\AppData\Local`,
		`SystemRoot=C:\Windows`,
		`HTTP_PROXY=http://192.0.2.1:9`,
	}

	env := make(map[string]string)
	for _, kv := range ChildEnv(inst, parent) {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			t.Fatalf("malformed child env entry %q", kv)
		}
		env[strings.ToUpper(key)] = value
	}

	if env["PATH"] != `C:\Windows\System32` {
		t.Fatalf("PATH = %q", env["PATH"])
	}
	if env["USERPROFILE"] != `C:\Users\fixture` {
		t.Fatalf("USERPROFILE = %q", env["USERPROFILE"])
	}
	wantPersonal := `C:\Users\fixture\.zcode\v2\provider_config.json`
	if env[childPersonalVar] != wantPersonal {
		t.Fatalf("%s = %q, want %q", childPersonalVar, env[childPersonalVar], wantPersonal)
	}
	if _, present := env["HTTP_PROXY"]; present {
		t.Fatal("proxy variable crossed the child allowlist")
	}
}

func TestWindowsMountUnsupported(t *testing.T) {
	_, err := PrepareMount(context.Background(), Install{AppImage: `C:\ZCode.AppImage`}, DefaultMountOpts())
	if !errors.Is(err, ErrAppImageUnsupported) {
		t.Fatalf("PrepareMount error = %v, want ErrAppImageUnsupported", err)
	}
	var nilMount *Mount
	if err := nilMount.Guard(); err != nil {
		t.Fatalf("nil mount guard = %v, want nil", err)
	}
}

func TestWindowsProcessTreeHelper(t *testing.T) {
	if os.Getenv(windowsTreeHelperEnv) != "1" {
		return
	}

	// The grandchild is this same Go test binary re-exec'd in sleep mode:
	// the process tree must not depend on cmd.exe or any other shell.
	cmd := exec.CommandContext(context.Background(), os.Args[0],
		"-test.run=^TestWindowsProcessTreeGrandchildHelper$")
	cmd.Env = append(os.Environ(), windowsTreeGrandchildHelperEnv+"=1")
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fmt.Fprintf(os.Stdout, "%d\n", cmd.Process.Pid)
	time.Sleep(time.Hour)
}

// TestWindowsProcessTreeGrandchildHelper is the re-exec'd grandchild: it
// stays alive until the Job Object terminates the whole tree.
func TestWindowsProcessTreeGrandchildHelper(t *testing.T) {
	if os.Getenv(windowsTreeGrandchildHelperEnv) != "1" {
		return
	}

	time.Sleep(time.Hour)
}

func TestWindowsProcessTreeTerminatesGrandchild(t *testing.T) {
	if os.Getenv(windowsTreeHelperEnv) == "1" {
		TestWindowsProcessTreeHelper(t)

		return
	}

	env := append(os.Environ(), windowsTreeHelperEnv+"=1")
	child, err := SpawnChild(context.Background(),
		[]string{os.Args[0], "-test.run=^TestWindowsProcessTreeHelper$"}, env, SpawnOpts{})
	if err != nil {
		t.Fatalf("spawn Windows tree helper: %v", err)
	}
	t.Cleanup(func() { _ = child.terminateTree() })

	line, err := bufio.NewReader(child.Stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("read grandchild pid: %v", err)
	}
	grandchildPID, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || grandchildPID <= 0 {
		t.Fatalf("grandchild pid line = %q, err = %v", line, err)
	}

	_ = child.Stdin.Close()
	if err := child.terminateTree(); err != nil {
		t.Fatalf("terminate Windows job object: %v", err)
	}

	select {
	case <-child.Done:
	case <-time.After(10 * time.Second):
		t.Fatal("Windows tree helper was not reaped")
	}

	deadline := time.Now().Add(5 * time.Second)
	for windowsProcessRunning(grandchildPID) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if windowsProcessRunning(grandchildPID) {
		t.Fatalf("grandchild pid %d survived Job Object termination", grandchildPID)
	}
}

func windowsProcessRunning(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return errors.Is(err, windows.ERROR_ACCESS_DENIED)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return true
	}

	const stillActive = 259

	return exitCode == stillActive
}
