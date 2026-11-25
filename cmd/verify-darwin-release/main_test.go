package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	fixtureArm64      = "arm64"
	fixtureAMD64      = "x86_64"
	memberListFailure = "archive member list is not exactly zcode-acp"
)

var (
	errFixtureContent = errors.New("unexpected fixture content")
	errLipoCalled     = errors.New("lipo must not run")
)

type tarMember struct {
	name     string
	content  string
	typeflag byte
	linkname string
}

func writeArchive(t *testing.T, archivePath string, members []tarMember) {
	t.Helper()
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, member := range members {
		typeflag := member.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{
			Name:     member.name,
			Mode:     0o755,
			Size:     int64(len(member.content)),
			Typeflag: typeflag,
			Linkname: member.linkname,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header %s: %v", member.name, err)
		}
		if _, err := tarWriter.Write([]byte(member.content)); err != nil {
			t.Fatalf("write tar member %s: %v", member.name, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
}

func archiveName(label string) string {
	return "zcode-acp_0.0.0-SNAPSHOT-test_" + label + ".tar.gz"
}

func defaultMembers(content string) []tarMember {
	return []tarMember{{name: expectedBinary, content: content}}
}

func createDist(t *testing.T, arm64Members []tarMember) string {
	t.Helper()
	dist := filepath.Join(t.TempDir(), "dist")
	if err := os.Mkdir(dist, 0o700); err != nil {
		t.Fatalf("create dist: %v", err)
	}

	writeArchive(t, filepath.Join(dist, archiveName("darwin_arm64")), arm64Members)
	writeArchive(t, filepath.Join(dist, archiveName("darwin_amd64")), defaultMembers(fixtureAMD64))
	writeArchive(t, filepath.Join(dist, archiveName("darwin_universal")), defaultMembers("universal"))
	writeManifest(t, dist, []string{
		archiveName("darwin_arm64"),
		archiveName("darwin_amd64"),
		archiveName("darwin_universal"),
	})

	return dist
}

func writeManifest(t *testing.T, dist string, names []string) {
	t.Helper()
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	lines := make([]string, 0, len(sorted))
	for _, name := range sorted {
		data, err := os.ReadFile(filepath.Join(dist, name))
		if err != nil {
			t.Fatalf("read archive for checksum: %v", err)
		}
		sum := sha256.Sum256(data)
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+name)
	}
	manifest := []byte(strings.Join(lines, "\n") + "\n")
	if err := os.WriteFile(filepath.Join(dist, checksumName), manifest, 0o600); err != nil {
		t.Fatalf("write checksum manifest: %v", err)
	}
}

func contentLipo(binary string) (string, error) {
	data, err := os.ReadFile(binary)
	if err != nil {
		return "", err
	}
	switch string(data) {
	case fixtureArm64:
		return fixtureArm64, nil
	case fixtureAMD64:
		return fixtureAMD64, nil
	case "universal":
		return fixtureAMD64 + " " + fixtureArm64, nil
	default:
		return "", fmt.Errorf("%w: %q", errFixtureContent, data)
	}
}

func runVerifier(t *testing.T, dist string, lipo lipoFunc) (code int, stdoutText, stderrText string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = run([]string{dist}, &stdout, &stderr, lipo)

	return code, stdout.String(), stderr.String()
}

func TestVerifierAcceptsValidArchives(t *testing.T) {
	dist := createDist(t, defaultMembers(fixtureArm64))

	code, stdout, stderr := runVerifier(t, dist, contentLipo)
	if code != 0 {
		t.Fatalf("verifier exit = %d, want 0; stderr:\n%s", code, stderr)
	}
	for _, marker := range []string{
		"PASS darwin_arm64: SHA-256, one regular zcode-acp, lipo -archs arm64",
		"PASS darwin_amd64: SHA-256, one regular zcode-acp, lipo -archs x86_64",
		"PASS darwin_universal: SHA-256, one regular zcode-acp, lipo -archs x86_64 arm64",
	} {
		if !strings.Contains(stdout, marker) {
			t.Fatalf("missing PASS marker %q; stdout:\n%s", marker, stdout)
		}
	}
}

type archiveRejectionCase struct {
	name       string
	members    []tarMember
	wantDetail string
}

func archiveRejectionCases() []archiveRejectionCase {
	return []archiveRejectionCase{
		{
			name:       "traversal",
			members:    []tarMember{{name: "../outside", content: "bad"}},
			wantDetail: memberListFailure,
		},
		{
			name:       "absolute",
			members:    []tarMember{{name: "/tmp/zcode-acp", content: "bad"}},
			wantDetail: memberListFailure,
		},
		{
			name:       "symlink",
			members:    []tarMember{{name: expectedBinary, typeflag: tar.TypeSymlink, linkname: "/etc/passwd"}},
			wantDetail: "archive member is not a regular file",
		},
		{
			name: "unexpected member",
			members: []tarMember{
				{name: expectedBinary, content: fixtureArm64},
				{name: "README.md", content: "extra"},
			},
			wantDetail: memberListFailure,
		},
		{
			name:       "missing binary",
			members:    []tarMember{{name: "README.md", content: "missing"}},
			wantDetail: memberListFailure,
		},
		{
			name:       "directory",
			members:    []tarMember{{name: expectedBinary, typeflag: tar.TypeDir}},
			wantDetail: "archive member is not a regular file",
		},
	}
}

func TestVerifierRejectsAdversarialArchivesBeforeLipo(t *testing.T) {
	for _, test := range archiveRejectionCases() {
		t.Run(test.name, func(t *testing.T) {
			dist := createDist(t, test.members)
			lipoCalls := 0
			lipo := func(string) (string, error) {
				lipoCalls++

				return "", errLipoCalled
			}

			code, stdout, stderr := runVerifier(t, dist, lipo)
			if code != 1 {
				t.Fatalf("verifier exit = %d, want 1; stdout:\n%s", code, stdout)
			}
			if !strings.Contains(stderr, test.wantDetail) {
				t.Fatalf("stderr missing %q:\n%s", test.wantDetail, stderr)
			}
			if lipoCalls != 0 {
				t.Fatalf("lipo ran %d times before archive rejection", lipoCalls)
			}
		})
	}
}

func TestVerifierRejectsChecksumAndManifestDrift(t *testing.T) {
	t.Run("archive corruption", func(t *testing.T) {
		dist := createDist(t, defaultMembers(fixtureArm64))
		archivePath := filepath.Join(dist, archiveName("darwin_arm64"))
		file, err := os.OpenFile(archivePath, os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatalf("open archive: %v", err)
		}
		if _, err := file.WriteString("corrupt"); err != nil {
			t.Fatalf("append archive: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close archive: %v", err)
		}

		code, _, stderr := runVerifier(t, dist, contentLipo)
		if code != 1 || !strings.Contains(stderr, "SHA-256 verification failed") {
			t.Fatalf("corrupt archive accepted: exit %d; stderr:\n%s", code, stderr)
		}
	})

	tests := []struct {
		name    string
		content string
	}{
		{name: "malformed", content: "not-a-checksum\n"},
		{name: "unexpected archive", content: strings.Repeat("a", 64) + "  other.tar.gz\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dist := createDist(t, defaultMembers(fixtureArm64))
			if err := os.WriteFile(filepath.Join(dist, checksumName), []byte(test.content), 0o600); err != nil {
				t.Fatalf("rewrite manifest: %v", err)
			}
			code, _, stderr := runVerifier(t, dist, contentLipo)
			if code != 1 {
				t.Fatalf("manifest drift accepted: stderr:\n%s", stderr)
			}
		})
	}
}

func TestVerifierRejectsWrongArchitectureSets(t *testing.T) {
	tests := []struct {
		name string
		lipo lipoFunc
	}{
		{
			name: "thin wrong arch",
			lipo: func(string) (string, error) { return fixtureAMD64, nil },
		},
		{
			name: "universal incomplete",
			lipo: func(binary string) (string, error) {
				data, err := os.ReadFile(binary)
				if err != nil {
					return "", err
				}
				if string(data) == "universal" {
					return fixtureArm64, nil
				}

				return contentLipo(binary)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dist := createDist(t, defaultMembers(fixtureArm64))
			code, _, stderr := runVerifier(t, dist, test.lipo)
			if code != 1 || !strings.Contains(stderr, "unexpected lipo architecture set") {
				t.Fatalf("wrong architecture accepted: exit %d; stderr:\n%s", code, stderr)
			}
		})
	}
}

func TestVerifierRejectsUnsafeInputsAndUsage(t *testing.T) {
	t.Run("usage", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := run(nil, &stdout, &stderr, contentLipo); code != 1 {
			t.Fatalf("usage exit = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "<dist-directory>") {
			t.Fatalf("usage text missing: %s", stderr.String())
		}
	})

	t.Run("symlink dist", func(t *testing.T) {
		dist := createDist(t, defaultMembers(fixtureArm64))
		link := filepath.Join(t.TempDir(), "dist-link")
		if err := os.Symlink(dist, link); err != nil {
			t.Fatalf("create dist symlink: %v", err)
		}
		code, _, stderr := runVerifier(t, link, contentLipo)
		if code != 1 || !strings.Contains(stderr, "not a non-symlink directory") {
			t.Fatalf("symlink dist accepted: exit %d; stderr:\n%s", code, stderr)
		}
	})

	t.Run("symlink archive", func(t *testing.T) {
		dist := createDist(t, defaultMembers(fixtureArm64))
		archivePath := filepath.Join(dist, archiveName("darwin_arm64"))
		target := filepath.Join(dist, archiveName("darwin_amd64"))
		if err := os.Remove(archivePath); err != nil {
			t.Fatalf("remove archive: %v", err)
		}
		if err := os.Symlink(target, archivePath); err != nil {
			t.Fatalf("create archive symlink: %v", err)
		}
		code, _, stderr := runVerifier(t, dist, contentLipo)
		if code != 1 || !strings.Contains(stderr, "archive is not a regular non-symlink file") {
			t.Fatalf("symlink archive accepted: exit %d; stderr:\n%s", code, stderr)
		}
	})
}
