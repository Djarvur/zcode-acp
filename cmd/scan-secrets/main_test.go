package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func newGitFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")

	return root
}

func writeSecretFixture(t *testing.T, root, relativePath string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", relativePath, err)
	}
}

func jwtFixture() string {
	return "ey" + "J" + strings.Repeat("a", 12) + "." + strings.Repeat("b", 8) + "." + strings.Repeat("c", 8)
}

func credentialFixture() string {
	return "enc:v" + "2:os:" + strings.Repeat("A", 24)
}

func tokenFixture() string {
	return `{"access_token":"` + strings.Repeat("t", 24) + `"}`
}

func TestScanDetectsAllCredentialShapesAndRedactsValues(t *testing.T) {
	root := newGitFixture(t)
	content := jwtFixture() + "\n" + credentialFixture() + "\n" + tokenFixture() + "\n"
	writeSecretFixture(t, root, "tracked.txt", []byte(content))
	runGit(t, root, "add", "tracked.txt")

	files, err := trackedFiles(root)
	if err != nil {
		t.Fatalf("trackedFiles: %v", err)
	}
	var out bytes.Buffer
	code := scanAt(root, files, &out)
	if code != 1 {
		t.Fatalf("scan exit = %d, want 1; output:\n%s", code, out.String())
	}
	output := out.String()
	for _, label := range []string{"eyJ", "enc:v[12]", "access_token"} {
		if !strings.Contains(output, label) {
			t.Fatalf("missing SECRET SHAPE label containing %q:\n%s", label, output)
		}
	}
	for _, secret := range []string{jwtFixture(), credentialFixture(), tokenFixture()} {
		if strings.Contains(output, secret) {
			t.Fatalf("scan leaked credential material %q:\n%s", secret, output)
		}
	}
	if count := strings.Count(output, "[REDACTED]"); count != 3 {
		t.Fatalf("[REDACTED] count = %d, want 3; output:\n%s", count, output)
	}
}

func TestTrackedScopeExcludesUntrackedAndPlanningFiles(t *testing.T) {
	root := newGitFixture(t)
	writeSecretFixture(t, root, "tracked.txt", []byte("clean\n"))
	writeSecretFixture(t, root, "untracked.txt", []byte(jwtFixture()+"\n"))
	writeSecretFixture(t, root, ".planning/leak.txt", []byte(jwtFixture()+"\n"))
	runGit(t, root, "add", "tracked.txt", ".planning/leak.txt")

	files, err := trackedFiles(root)
	if err != nil {
		t.Fatalf("trackedFiles: %v", err)
	}
	if strings.Contains(strings.Join(files, "\n"), "untracked.txt") {
		t.Fatalf("untracked file entered tracked scope: %v", files)
	}
	if strings.Contains(strings.Join(files, "\n"), ".planning/leak.txt") {
		t.Fatalf(".planning file entered tracked scope: %v", files)
	}

	var out bytes.Buffer
	if code := scanAt(root, files, &out); code != 0 {
		t.Fatalf("excluded secret caused failure: exit %d; output:\n%s", code, out.String())
	}
}

func TestAllowlistedSyntheticCredentialMarkerIsIgnored(t *testing.T) {
	root := t.TempDir()
	marker := allowedLines()[4]
	writeSecretFixture(t, root, "fixture.txt", []byte(`value = "`+marker+`"`+"\n"))

	var out bytes.Buffer
	if code := scanAt(root, []string{"fixture.txt"}, &out); code != 0 {
		t.Fatalf("allowlisted marker failed: exit %d; output:\n%s", code, out.String())
	}
}

func TestCleanProseAndShortValuesPass(t *testing.T) {
	root := t.TempDir()
	content := "enc:v2:os: prefix documentation only\n" +
		`{"apiKey":"short"}` + "\n" +
		"A three-part JWT has two dots.\n"
	writeSecretFixture(t, root, "clean.txt", []byte(content))

	var out bytes.Buffer
	if code := scanAt(root, []string{"clean.txt"}, &out); code != 0 {
		t.Fatalf("clean content failed: exit %d; output:\n%s", code, out.String())
	}
}

func TestBinaryMatchesNeverPrintSecretMaterial(t *testing.T) {
	root := t.TempDir()
	data := append([]byte{0}, []byte(jwtFixture())...)
	writeSecretFixture(t, root, "binary.dat", data)

	var out bytes.Buffer
	if code := scanAt(root, []string{"binary.dat"}, &out); code != 1 {
		t.Fatalf("binary match exit = %d, want 1; output:\n%s", code, out.String())
	}
	output := out.String()
	if !strings.Contains(output, "Binary file binary.dat matches") {
		t.Fatalf("binary finding was not reported safely:\n%s", output)
	}
	if strings.Contains(output, jwtFixture()) {
		t.Fatalf("binary finding leaked credential material:\n%s", output)
	}
}
