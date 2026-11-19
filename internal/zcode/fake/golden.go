package fake

import (
	"bytes"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// UpdateFlagName is the test-binary flag that (re)writes golden fixtures
// from the current run; register it in TestMain and pass -update.
const UpdateFlagName = "update"

// Golden-fixture permissions. The perms keep gosec quiet: fixtures are
// repo-controlled test data, never secrets.
const (
	goldenDirPerm  = 0o750
	goldenFilePerm = 0o600
)

// CompareGolden asserts got against the golden fixture at path. With the
// -update flag (registered by the package's TestMain) the fixture is
// (re)written from got — recording is explicit. Without it a missing
// fixture FAILS the test: missing frames are never normalized away and a
// gap is never silently auto-created (record-honesty discipline).
func CompareGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if updateRequested() {
		if err := os.MkdirAll(filepath.Dir(path), goldenDirPerm); err != nil {
			t.Fatalf("create golden dir: %v", err)
		}

		if err := os.WriteFile(path, got, goldenFilePerm); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		t.Logf("golden updated: %s", path)

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("golden fixture missing (record it with -update): %s", path)
		}

		t.Fatalf("read golden: %v", err)
	}

	if !bytes.Equal(want, got) {
		t.Fatalf("golden drift at %s:\n--- want ---\n%s--- got ---\n%s", path, want, got)
	}
}

// updateRequested reports whether the test binary was given -update.
func updateRequested() bool {
	f := flag.Lookup(UpdateFlagName)

	return f != nil && f.Value.String() == "true"
}
