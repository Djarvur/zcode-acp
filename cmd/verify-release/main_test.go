package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"debug/elf"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	fixtureMarker     = "18.1.9+zacp.test"
	fixtureAMD64      = "x86_64"
	fixtureARM64      = "aarch64"
	fixtureLipoARM64  = "arm64"
	fixtureUniversal  = "universal"
	fixtureBadContent = "bad"
	fixtureNotRegular = "archive member is not a regular file"
	fixtureReadmeName = "README.md"
	fixtureMissing    = "missing"
	memberListPrefix  = "archive member list is not exactly "
	peHeaderOffset    = 0x3c
	fixtureFileMode   = 0o600
	fixtureExecutable = 0o755
)

type fixtureMember struct {
	name     string
	content  string
	typeflag byte
	linkname string
}

func fixtureContent(arch string) string {
	return arch + "\n" + fixtureMarker
}

func archiveName(spec archiveSpec) string {
	return "zcode-acp_0.0.0-SNAPSHOT-test_" + spec.label + spec.extension
}

func defaultArchives() map[string][]fixtureMember {
	return map[string][]fixtureMember{
		"linux_amd64":      {{name: expectedBinary, content: fixtureContent(fixtureAMD64)}},
		"linux_arm64":      {{name: expectedBinary, content: fixtureContent(fixtureARM64)}},
		"darwin_amd64":     {{name: expectedBinary, content: fixtureContent(fixtureAMD64)}},
		"darwin_arm64":     {{name: expectedBinary, content: fixtureContent(fixtureLipoARM64)}},
		"darwin_universal": {{name: expectedBinary, content: fixtureContent(fixtureUniversal)}},
		"windows_amd64":    {{name: windowsBinary, content: fixtureContent(fixtureAMD64)}},
	}
}

func createDist(t *testing.T, archives map[string][]fixtureMember) string {
	t.Helper()

	dist := filepath.Join(t.TempDir(), "dist")
	if err := os.Mkdir(dist, 0o700); err != nil {
		t.Fatalf("create dist: %v", err)
	}

	names := make([]string, 0, len(archives))
	for _, spec := range archiveSpecs() {
		members, ok := archives[spec.label]
		if !ok {
			continue
		}
		name := archiveName(spec)
		writeFixtureArchive(t, filepath.Join(dist, name), spec, members)
		names = append(names, name)
	}
	writeManifest(t, dist, names)

	return dist
}

func writeFixtureArchive(t *testing.T, archivePath string, spec archiveSpec, members []fixtureMember) {
	t.Helper()
	if spec.extension == archiveExtensionZip {
		writeZipArchive(t, archivePath, members)

		return
	}
	writeTarGzArchive(t, archivePath, members)
}

func writeTarGzArchive(t *testing.T, archivePath string, members []fixtureMember) {
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
			Mode:     fixtureExecutable,
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

func writeZipArchive(t *testing.T, archivePath string, members []fixtureMember) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create zip archive: %v", err)
	}
	writer := zip.NewWriter(file)
	for _, member := range members {
		header := &zip.FileHeader{Name: member.name, Method: zip.Deflate}
		switch member.typeflag {
		case tar.TypeSymlink:
			header.SetMode(os.ModeSymlink | fixtureExecutable)
		case tar.TypeDir:
			header.SetMode(os.ModeDir | fixtureExecutable)
		default:
			header.SetMode(fixtureExecutable)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create zip member %s: %v", member.name, err)
		}
		if _, err := entry.Write([]byte(member.content)); err != nil {
			t.Fatalf("write zip member %s: %v", member.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip archive: %v", err)
	}
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

func firstLine(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	line, _, _ := strings.Cut(string(data), "\n")

	return line, nil
}

func fixtureLipo(filePath string) (string, error) {
	arch, err := firstLine(filePath)
	if err != nil {
		return "", err
	}
	if arch == fixtureUniversal {
		return fixtureAMD64 + " " + fixtureLipoARM64, nil
	}

	return arch, nil
}

func fixtureELF(filePath string) (string, error) {
	return firstLine(filePath)
}

func fixturePE(filePath string) (string, error) {
	return firstLine(filePath)
}

func fixtureInspectors() inspectors {
	return inspectors{lipo: fixtureLipo, elf: fixtureELF, pe: fixturePE}
}

type inspectorCounts struct {
	lipo int
	elf  int
	pe   int
}

func countingInspectors(counts *inspectorCounts) inspectors {
	return inspectors{
		lipo: func(binary string) (string, error) {
			counts.lipo++

			return fixtureLipo(binary)
		},
		elf: func(binary string) (string, error) {
			counts.elf++

			return fixtureELF(binary)
		},
		pe: func(binary string) (string, error) {
			counts.pe++

			return fixturePE(binary)
		},
	}
}

func runVerifier(t *testing.T, dist string, ins inspectors) (code int, stdoutText, stderrText string) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code = run([]string{dist}, &stdout, &stderr, ins)

	return code, stdout.String(), stderr.String()
}

func TestVerifierAcceptsValidPlatformArchives(t *testing.T) {
	dist := createDist(t, defaultArchives())

	code, stdout, stderr := runVerifier(t, dist, fixtureInspectors())
	if code != 0 {
		t.Fatalf("verifier exit = %d, want 0; stderr:\n%s", code, stderr)
	}
	for _, marker := range []string{
		"PASS linux_amd64: SHA-256, one regular zcode-acp, ELF x86_64, version stamp 18.1.9+zacp.",
		"PASS linux_arm64: SHA-256, one regular zcode-acp, ELF aarch64, version stamp 18.1.9+zacp.",
		"PASS darwin_amd64: SHA-256, one regular zcode-acp, lipo -archs x86_64, version stamp 18.1.9+zacp.",
		"PASS darwin_arm64: SHA-256, one regular zcode-acp, lipo -archs arm64, version stamp 18.1.9+zacp.",
		"PASS darwin_universal: SHA-256, one regular zcode-acp, lipo -archs x86_64 arm64, version stamp 18.1.9+zacp.",
		"PASS windows_amd64: SHA-256, one regular zcode-acp.exe, PE x86_64, version stamp 18.1.9+zacp.",
	} {
		if !strings.Contains(stdout, marker) {
			t.Fatalf("missing PASS marker %q; stdout:\n%s", marker, stdout)
		}
	}
}

type archiveRejectionCase struct {
	name       string
	members    []fixtureMember
	wantDetail string
}

func tarRejectionCases() []archiveRejectionCase {
	return []archiveRejectionCase{
		{
			name:       "traversal",
			members:    []fixtureMember{{name: "../outside", content: fixtureBadContent}},
			wantDetail: memberListPrefix + expectedBinary,
		},
		{
			name:       "absolute",
			members:    []fixtureMember{{name: "/tmp/zcode-acp", content: fixtureBadContent}},
			wantDetail: memberListPrefix + expectedBinary,
		},
		{
			name:       "symlink",
			members:    []fixtureMember{{name: expectedBinary, typeflag: tar.TypeSymlink, linkname: "/etc/passwd"}},
			wantDetail: fixtureNotRegular,
		},
		{
			name: "unexpected member",
			members: []fixtureMember{
				{name: expectedBinary, content: fixtureContent(fixtureAMD64)},
				{name: fixtureReadmeName, content: "extra"},
			},
			wantDetail: memberListPrefix + expectedBinary,
		},
		{
			name:       "missing binary",
			members:    []fixtureMember{{name: fixtureReadmeName, content: fixtureMissing}},
			wantDetail: memberListPrefix + expectedBinary,
		},
		{
			name:       "directory",
			members:    []fixtureMember{{name: expectedBinary, typeflag: tar.TypeDir}},
			wantDetail: fixtureNotRegular,
		},
	}
}

func zipRejectionCases() []archiveRejectionCase {
	return []archiveRejectionCase{
		{
			name:       "traversal",
			members:    []fixtureMember{{name: "../outside.exe", content: fixtureBadContent}},
			wantDetail: memberListPrefix + windowsBinary,
		},
		{
			name:       "absolute",
			members:    []fixtureMember{{name: "/tmp/zcode-acp.exe", content: fixtureBadContent}},
			wantDetail: memberListPrefix + windowsBinary,
		},
		{
			name: "symlink",
			members: []fixtureMember{
				{name: windowsBinary, typeflag: tar.TypeSymlink, linkname: `C:\Windows\System32\cmd.exe`},
			},
			wantDetail: fixtureNotRegular,
		},
		{
			name: "unexpected member",
			members: []fixtureMember{
				{name: windowsBinary, content: fixtureContent(fixtureAMD64)},
				{name: fixtureReadmeName, content: "extra"},
			},
			wantDetail: memberListPrefix + windowsBinary,
		},
		{
			name:       "missing binary",
			members:    []fixtureMember{{name: fixtureReadmeName, content: fixtureMissing}},
			wantDetail: memberListPrefix + windowsBinary,
		},
		{
			name:       "directory",
			members:    []fixtureMember{{name: windowsBinary, typeflag: tar.TypeDir}},
			wantDetail: fixtureNotRegular,
		},
	}
}

func TestVerifierRejectsAdversarialTarArchivesBeforeInspection(t *testing.T) {
	for _, test := range tarRejectionCases() {
		t.Run(test.name, func(t *testing.T) {
			archives := defaultArchives()
			archives["linux_amd64"] = test.members
			dist := createDist(t, archives)
			counts := &inspectorCounts{}

			code, stdout, stderr := runVerifier(t, dist, countingInspectors(counts))
			if code != 1 {
				t.Fatalf("verifier exit = %d, want 1; stdout:\n%s", code, stdout)
			}
			if !strings.Contains(stderr, test.wantDetail) {
				t.Fatalf("stderr missing %q:\n%s", test.wantDetail, stderr)
			}
			if counts.lipo+counts.elf+counts.pe != 0 {
				t.Fatalf("architecture inspection ran before archive rejection: %+v", counts)
			}
		})
	}
}

func TestVerifierRejectsAdversarialZipArchivesBeforeInspection(t *testing.T) {
	for _, test := range zipRejectionCases() {
		t.Run(test.name, func(t *testing.T) {
			archives := defaultArchives()
			archives["windows_amd64"] = test.members
			dist := createDist(t, archives)
			counts := &inspectorCounts{}

			code, stdout, stderr := runVerifier(t, dist, countingInspectors(counts))
			if code != 1 {
				t.Fatalf("verifier exit = %d, want 1; stdout:\n%s", code, stdout)
			}
			if !strings.Contains(stderr, test.wantDetail) {
				t.Fatalf("stderr missing %q:\n%s", test.wantDetail, stderr)
			}
			if counts.pe != 0 {
				t.Fatalf("PE inspection ran before zip archive rejection: %+v", counts)
			}
		})
	}
}

func TestVerifierRejectsChecksumAndManifestDrift(t *testing.T) {
	t.Run("archive corruption", func(t *testing.T) {
		dist := createDist(t, defaultArchives())
		archivePath := filepath.Join(dist, archiveName(archiveSpecs()[0]))
		file, err := os.OpenFile(archivePath, os.O_APPEND|os.O_WRONLY, fixtureFileMode)
		if err != nil {
			t.Fatalf("open archive: %v", err)
		}
		if _, err := file.WriteString("corrupt"); err != nil {
			t.Fatalf("append archive: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close archive: %v", err)
		}

		code, _, stderr := runVerifier(t, dist, fixtureInspectors())
		if code != 1 || !strings.Contains(stderr, "SHA-256 verification failed") {
			t.Fatalf("corrupt archive accepted: exit %d; stderr:\n%s", code, stderr)
		}
	})

	t.Run("malformed manifest", func(t *testing.T) {
		dist := createDist(t, defaultArchives())
		manifest := []byte("not-a-checksum\n")
		if err := os.WriteFile(filepath.Join(dist, checksumName), manifest, fixtureFileMode); err != nil {
			t.Fatalf("rewrite manifest: %v", err)
		}
		code, _, stderr := runVerifier(t, dist, fixtureInspectors())
		if code != 1 || !strings.Contains(stderr, "malformed SHA-256 checksum line") {
			t.Fatalf("malformed manifest accepted: exit %d; stderr:\n%s", code, stderr)
		}
	})

	t.Run("unexpected archive", func(t *testing.T) {
		dist := createDist(t, defaultArchives())
		content := strings.Repeat("a", 64) + "  other.tar.gz\n"
		if err := os.WriteFile(filepath.Join(dist, checksumName), []byte(content), fixtureFileMode); err != nil {
			t.Fatalf("rewrite manifest: %v", err)
		}
		code, _, stderr := runVerifier(t, dist, fixtureInspectors())
		if code != 1 || !strings.Contains(stderr, "not an expected release archive") {
			t.Fatalf("unexpected archive accepted: exit %d; stderr:\n%s", code, stderr)
		}
	})

	t.Run("partial manifest", func(t *testing.T) {
		dist := createDist(t, defaultArchives())
		names := make([]string, 0, 5)
		for _, spec := range archiveSpecs()[:5] {
			names = append(names, archiveName(spec))
		}
		writeManifest(t, dist, names)
		code, _, stderr := runVerifier(t, dist, fixtureInspectors())
		if code != 1 || !strings.Contains(stderr, "must cover exactly the six release archives") {
			t.Fatalf("partial manifest accepted: exit %d; stderr:\n%s", code, stderr)
		}
	})
}

type wrongArchitectureCase struct {
	name       string
	inspectors inspectors
	wantLabel  string
}

func wrongArchitectureCases() []wrongArchitectureCase {
	return []wrongArchitectureCase{
		{
			name: "linux thin",
			inspectors: inspectors{
				lipo: fixtureLipo,
				elf:  func(string) (string, error) { return fixtureARM64, nil },
				pe:   fixturePE,
			},
			wantLabel: "linux_amd64",
		},
		{
			name: "darwin thin",
			inspectors: inspectors{
				lipo: func(string) (string, error) { return fixtureLipoARM64, nil },
				elf:  fixtureELF,
				pe:   fixturePE,
			},
			wantLabel: "darwin_amd64",
		},
		{
			name: "darwin universal incomplete",
			inspectors: inspectors{
				lipo: func(filePath string) (string, error) {
					data, err := os.ReadFile(filePath)
					if err != nil {
						return "", err
					}
					if string(data) == fixtureContent(fixtureUniversal) {
						return fixtureLipoARM64, nil
					}

					return fixtureLipo(filePath)
				},
				elf: fixtureELF,
				pe:  fixturePE,
			},
			wantLabel: "darwin_universal",
		},
		{
			name: "windows",
			inspectors: inspectors{
				lipo: fixtureLipo,
				elf:  fixtureELF,
				pe:   func(string) (string, error) { return fixtureARM64, nil },
			},
			wantLabel: "windows_amd64",
		},
	}
}

func TestVerifierRejectsWrongArchitectureSets(t *testing.T) {
	for _, test := range wrongArchitectureCases() {
		t.Run(test.name, func(t *testing.T) {
			dist := createDist(t, defaultArchives())

			code, _, stderr := runVerifier(t, dist, test.inspectors)
			wantDetail := test.wantLabel + " has unexpected architecture set"
			if code != 1 || !strings.Contains(stderr, wantDetail) {
				t.Fatalf("wrong architecture accepted: exit %d; stderr:\n%s", code, stderr)
			}
		})
	}
}

func TestVerifierRejectsMissingVersionStamp(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "missing", content: fixtureARM64},
		{name: "wrong build metadata", content: fixtureARM64 + "\n18.1.9+zacpX"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archives := defaultArchives()
			archives["linux_arm64"] = []fixtureMember{{name: expectedBinary, content: test.content}}
			dist := createDist(t, archives)

			code, _, stderr := runVerifier(t, dist, fixtureInspectors())
			wantDetail := "binary does not carry the 18.1.9+zacp. version stamp"
			if code != 1 || !strings.Contains(stderr, wantDetail) {
				t.Fatalf("missing version stamp accepted: exit %d; stderr:\n%s", code, stderr)
			}
		})
	}
}

func TestVerifierRejectsUnsafeInputsAndUsage(t *testing.T) {
	t.Run("usage", testVerifierUsage)
	t.Run("symlink dist", testVerifierRejectsSymlinkDist)
	t.Run("symlink archive", testVerifierRejectsSymlinkArchive)
	t.Run("missing archive", testVerifierRejectsMissingArchive)
	t.Run("duplicate archive", testVerifierRejectsDuplicateArchive)
}

func testVerifierUsage(t *testing.T) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr, fixtureInspectors()); code != 1 {
		t.Fatalf("usage exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "<dist-directory>") {
		t.Fatalf("usage text missing: %s", stderr.String())
	}
}

func testVerifierRejectsSymlinkDist(t *testing.T) {
	t.Helper()

	dist := createDist(t, defaultArchives())
	link := filepath.Join(t.TempDir(), "dist-link")
	if err := os.Symlink(dist, link); err != nil {
		t.Fatalf("create dist symlink: %v", err)
	}
	code, _, stderr := runVerifier(t, link, fixtureInspectors())
	if code != 1 || !strings.Contains(stderr, "not a non-symlink directory") {
		t.Fatalf("symlink dist accepted: exit %d; stderr:\n%s", code, stderr)
	}
}

func testVerifierRejectsSymlinkArchive(t *testing.T) {
	t.Helper()

	dist := createDist(t, defaultArchives())
	archivePath := filepath.Join(dist, archiveName(archiveSpecs()[0]))
	target := filepath.Join(dist, archiveName(archiveSpecs()[1]))
	if err := os.Remove(archivePath); err != nil {
		t.Fatalf("remove archive: %v", err)
	}
	if err := os.Symlink(target, archivePath); err != nil {
		t.Fatalf("create archive symlink: %v", err)
	}
	code, _, stderr := runVerifier(t, dist, fixtureInspectors())
	if code != 1 || !strings.Contains(stderr, "archive is not a regular non-symlink file") {
		t.Fatalf("symlink archive accepted: exit %d; stderr:\n%s", code, stderr)
	}
}

func testVerifierRejectsMissingArchive(t *testing.T) {
	t.Helper()

	dist := createDist(t, defaultArchives())
	if err := os.Remove(filepath.Join(dist, archiveName(archiveSpecs()[0]))); err != nil {
		t.Fatalf("remove archive: %v", err)
	}
	code, _, stderr := runVerifier(t, dist, fixtureInspectors())
	wantDetail := "expected exactly one linux_amd64 archive, found 0"
	if code != 1 || !strings.Contains(stderr, wantDetail) {
		t.Fatalf("missing archive accepted: exit %d; stderr:\n%s", code, stderr)
	}
}

func testVerifierRejectsDuplicateArchive(t *testing.T) {
	t.Helper()

	dist := createDist(t, defaultArchives())
	source := filepath.Join(dist, archiveName(archiveSpecs()[0]))
	duplicate := filepath.Join(dist, "zcode-acp_extra_linux_amd64.tar.gz")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if err := os.WriteFile(duplicate, data, fixtureFileMode); err != nil {
		t.Fatalf("write duplicate archive: %v", err)
	}
	code, _, stderr := runVerifier(t, dist, fixtureInspectors())
	wantDetail := "expected exactly one linux_amd64 archive, found 2"
	if code != 1 || !strings.Contains(stderr, wantDetail) {
		t.Fatalf("duplicate archive accepted: exit %d; stderr:\n%s", code, stderr)
	}
}

func writeMinimalELF(t *testing.T, path string, class elf.Class, dataEncoding elf.Data, machine elf.Machine) {
	t.Helper()

	headerSize := 52
	if class == elf.ELFCLASS64 {
		headerSize = 64
	}
	content := make([]byte, headerSize+len(fixtureMarker))
	copy(content, "\x7fELF")
	content[elf.EI_CLASS] = byte(class)
	content[elf.EI_DATA] = byte(dataEncoding)
	content[elf.EI_VERSION] = byte(elf.EV_CURRENT)

	byteOrder := binary.ByteOrder(binary.LittleEndian)
	if dataEncoding == elf.ELFDATA2MSB {
		byteOrder = binary.BigEndian
	}
	byteOrder.PutUint16(content[16:], uint16(elf.ET_EXEC))
	byteOrder.PutUint16(content[18:], uint16(machine))
	byteOrder.PutUint32(content[20:], uint32(elf.EV_CURRENT))
	copy(content[headerSize:], fixtureMarker)

	if err := os.WriteFile(path, content, fixtureFileMode); err != nil {
		t.Fatalf("write ELF fixture: %v", err)
	}
}

func TestELFInspectorRecognizesSupportedTargets(t *testing.T) {
	tests := []struct {
		name    string
		machine elf.Machine
		want    string
	}{
		{name: "amd64", machine: elf.EM_X86_64, want: fixtureAMD64},
		{name: "arm64", machine: elf.EM_AARCH64, want: fixtureARM64},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), expectedBinary)
			writeMinimalELF(t, path, elf.ELFCLASS64, elf.ELFDATA2LSB, test.machine)

			got, err := elfArch(path)
			if err != nil {
				t.Fatalf("elfArch: %v", err)
			}
			if got != test.want {
				t.Fatalf("elfArch = %q, want %q", got, test.want)
			}
		})
	}
}

func TestELFInspectorRejectsUnsupportedInputs(t *testing.T) {
	tests := []struct {
		name       string
		write      func(t *testing.T, path string)
		wantDetail string
	}{
		{
			name: "32-bit class",
			write: func(t *testing.T, path string) {
				t.Helper()
				writeMinimalELF(t, path, elf.ELFCLASS32, elf.ELFDATA2LSB, elf.EM_X86_64)
			},
			wantDetail: "unexpected ELF class",
		},
		{
			name: "big-endian encoding",
			write: func(t *testing.T, path string) {
				t.Helper()
				writeMinimalELF(t, path, elf.ELFCLASS64, elf.ELFDATA2MSB, elf.EM_X86_64)
			},
			wantDetail: "unexpected ELF data encoding",
		},
		{
			name: "unsupported machine",
			write: func(t *testing.T, path string) {
				t.Helper()
				writeMinimalELF(t, path, elf.ELFCLASS64, elf.ELFDATA2LSB, elf.EM_ARM)
			},
			wantDetail: "unexpected ELF machine",
		},
		{
			name: "not an ELF binary",
			write: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("not an ELF binary"), fixtureFileMode); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
			},
			wantDetail: "open ELF binary",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), expectedBinary)
			test.write(t, path)

			if _, err := elfArch(path); err == nil || !strings.Contains(err.Error(), test.wantDetail) {
				t.Fatalf("elfArch error = %v, want %q", err, test.wantDetail)
			}
		})
	}
}

func writeMinimalPE(t *testing.T, path string, machine, characteristics uint16) {
	t.Helper()

	const (
		dosHeaderSize   = 96
		peSignatureSize = 4
		fileHeaderSize  = 20
	)
	markerOffset := dosHeaderSize + peSignatureSize + fileHeaderSize
	content := make([]byte, markerOffset+len(fixtureMarker))
	content[0] = 'M'
	content[1] = 'Z'
	binary.LittleEndian.PutUint32(content[peHeaderOffset:], dosHeaderSize)
	copy(content[dosHeaderSize:], "PE\x00\x00")
	binary.LittleEndian.PutUint16(content[dosHeaderSize+peSignatureSize:], machine)
	binary.LittleEndian.PutUint16(content[dosHeaderSize+peSignatureSize+fileHeaderSize-2:], characteristics)
	copy(content[markerOffset:], fixtureMarker)

	if err := os.WriteFile(path, content, fixtureFileMode); err != nil {
		t.Fatalf("write PE fixture: %v", err)
	}
}

func TestPEInspectorRecognizesWindowsAmd64(t *testing.T) {
	path := filepath.Join(t.TempDir(), windowsBinary)
	writeMinimalPE(t, path, pe.IMAGE_FILE_MACHINE_AMD64, pe.IMAGE_FILE_EXECUTABLE_IMAGE)

	got, err := peArch(path)
	if err != nil {
		t.Fatalf("peArch: %v", err)
	}
	if got != fixtureAMD64 {
		t.Fatalf("peArch = %q, want %q", got, fixtureAMD64)
	}
}

func TestPEInspectorRejectsUnsupportedInputs(t *testing.T) {
	tests := []struct {
		name            string
		machine         uint16
		characteristics uint16
		write           func(t *testing.T, path string)
		wantDetail      string
	}{
		{
			name:            "arm64 machine",
			machine:         pe.IMAGE_FILE_MACHINE_ARM64,
			characteristics: pe.IMAGE_FILE_EXECUTABLE_IMAGE,
			wantDetail:      "unexpected PE machine",
		},
		{
			name:       "not an executable image",
			machine:    pe.IMAGE_FILE_MACHINE_AMD64,
			wantDetail: "not an executable image",
		},
		{
			name: "not a PE binary",
			write: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("not a PE binary"), fixtureFileMode); err != nil {
					t.Fatalf("write fixture: %v", err)
				}
			},
			wantDetail: "open PE binary",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), windowsBinary)
			if test.write != nil {
				test.write(t, path)
			} else {
				writeMinimalPE(t, path, test.machine, test.characteristics)
			}

			if _, err := peArch(path); err == nil || !strings.Contains(err.Error(), test.wantDetail) {
				t.Fatalf("peArch error = %v, want %q", err, test.wantDetail)
			}
		})
	}
}
