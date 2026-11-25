//go:build linux

// In-package discovery seam test (WR-01): defaultDpkgQuery is unexported, so
// the production dpkg status parse is exercised here against fake dpkg-query
// and dpkg binaries on PATH. The external test package stubs the seam away,
// which is exactly why the off-by-one status match escaped the suite.

package supervisor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultDpkgQuery pins the production seam's status parse against the
// REAL dpkg-query output shape: `dpkg-query -W -f='${Status}'` prints
// "install ok installed" with no leading space, so a space-prefixed substring
// can never match. The installed arm must select the file list; the rc-state
// remnant must fall through without ever running dpkg -L.
func TestDefaultDpkgQuery(t *testing.T) {
	t.Run("installed status returns the file list", testDefaultDpkgQueryInstalled)
	t.Run("rc-state remnant falls through without the file list", testDefaultDpkgQueryRemnant)
}

// testDefaultDpkgQueryInstalled proves the exact no-leading-space status
// output installs the package.
func testDefaultDpkgQueryInstalled(t *testing.T) {
	binDir := t.TempDir()
	writeFakeCommand(t, binDir, "dpkg-query", `printf '%s' 'install ok installed'`)
	writeFakeCommand(t, binDir, "dpkg",
		`printf '/opt/ZCode/resources/glm/zcode.cjs\n/usr/bin/zcode\n'`)

	t.Setenv("PATH", binDir)

	out, err := defaultDpkgQuery(context.Background())
	if err != nil {
		t.Fatalf("defaultDpkgQuery for an installed package: %v", err)
	}

	for _, want := range []string{"/opt/ZCode/resources/glm/zcode.cjs", "/usr/bin/zcode"} {
		if !strings.Contains(out, want) {
			t.Fatalf("the installed-package file list %q must carry %q", out, want)
		}
	}
}

// testDefaultDpkgQueryRemnant proves a removed-but-not-purged package answers
// an empty list and never runs dpkg -L.
func testDefaultDpkgQueryRemnant(t *testing.T) {
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "dpkg-list-ran")
	writeFakeCommand(t, binDir, "dpkg-query", `printf '%s' 'deinstall ok config-files'`)
	writeFakeCommand(t, binDir, "dpkg", `: > '`+marker+`'`)

	t.Setenv("PATH", binDir)

	out, err := defaultDpkgQuery(context.Background())
	if err != nil {
		t.Fatalf("an rc-state remnant must not be an error: %v", err)
	}

	if out != "" {
		t.Fatalf("an rc-state remnant must answer an empty file list, got %q", out)
	}

	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("the rc-state remnant still ran dpkg -L (marker %s: %v)", marker, statErr)
	}
}

// writeFakeCommand writes one executable fake command into dir.
func writeFakeCommand(t *testing.T, dir, name, body string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o750); err != nil {
		t.Fatalf("write fake %s: %v", name, err)
	}
}
