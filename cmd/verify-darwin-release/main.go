// Command verify-darwin-release verifies a GoReleaser macOS snapshot without
// executing any release binary. It replaces the former shell verifier with
// archive/tar, compress/gzip, crypto/sha256, and a lipo subprocess.
package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	expectedBinary       = "zcode-acp"
	checksumName         = "checksums.txt"
	maxChecksumLineBytes = 1024 * 1024
	initialScanBuffer    = 64 * 1024
	privateDirMode       = 0o700
	privateFileMode      = 0o600
	maxArtifactBytes     = 256 * 1024 * 1024
)

var errVerify = errors.New("verify-darwin-release")

type archiveSpec struct {
	label     string
	archs     []string
	universal bool
}

type checksumEntry struct {
	name   string
	digest string
}

type lipoFunc func(binary string) (string, error)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, lipoArchs))
}

func run(args []string, stdout, stderr io.Writer, lipo lipoFunc) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintf(stderr, "%v: usage: %s <dist-directory>\n", errVerify, filepath.Base(os.Args[0]))

		return 1
	}
	if err := verifyDist(args[0], stdout, lipo); err != nil {
		_, _ = fmt.Fprintln(stderr, err)

		return 1
	}

	return 0
}

func verifyDist(dist string, out io.Writer, lipo lipoFunc) error {
	resolved, err := resolveDist(dist)
	if err != nil {
		return err
	}

	specs := archiveSpecs()
	archives, expectedNames, err := locateArchives(resolved, specs)
	if err != nil {
		return err
	}

	manifest := filepath.Join(resolved, checksumName)
	if err := requireRegularFile(manifest, "checksums.txt"); err != nil {
		return err
	}
	entries, err := readChecksums(manifest, expectedNames)
	if err != nil {
		return err
	}
	if err := verifyChecksums(resolved, entries); err != nil {
		return err
	}

	stage, err := os.MkdirTemp("", "zcode-acp-darwin-release-*")
	if err != nil {
		return fmt.Errorf("%w: create staging directory: %w", errVerify, err)
	}
	defer func() { _ = os.RemoveAll(stage) }()

	stagedArchives, err := stageArchives(stage, manifest, archives)
	if err != nil {
		return err
	}
	archiveDir := filepath.Join(stage, "archives")
	if err := verifyChecksums(archiveDir, entries); err != nil {
		return failf("staged archive checksum verification failed")
	}

	for _, spec := range specs {
		if err := verifyArchive(stagedArchives[spec.label], spec, stage, out, lipo); err != nil {
			return err
		}
	}

	return nil
}

func resolveDist(dist string) (string, error) {
	//nolint:gosec // the caller-supplied dist path is the subject of this validation.
	info, err := os.Lstat(dist)
	if err != nil {
		return "", failf("dist path is not a non-symlink directory: %s", dist)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", failf("dist path is not a non-symlink directory: %s", dist)
	}
	if !info.IsDir() {
		return "", failf("dist path is not a non-symlink directory: %s", dist)
	}

	resolved, err := filepath.EvalSymlinks(dist)
	if err != nil {
		return "", fmt.Errorf("%w: resolve dist path: %w", errVerify, err)
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("%w: make dist path absolute: %w", errVerify, err)
	}

	return absolute, nil
}

func locateArchives(
	dist string,
	specs []archiveSpec,
) (archives map[string]string, expectedNames map[string]struct{}, err error) {
	archives = make(map[string]string, len(specs))
	expectedNames = make(map[string]struct{}, len(specs))

	for _, spec := range specs {
		pattern := filepath.Join(dist, "zcode-acp_*_"+spec.label+".tar.gz")
		matches, globErr := filepath.Glob(pattern)
		if globErr != nil {
			return nil, nil, fmt.Errorf("%w: expand %s archive pattern: %w", errVerify, spec.label, globErr)
		}
		if len(matches) != 1 {
			return nil, nil, failf("expected exactly one %s archive, found %d", spec.label, len(matches))
		}
		if err := requireRegularFile(matches[0], "archive"); err != nil {
			return nil, nil, err
		}
		archives[spec.label] = matches[0]
		expectedNames[filepath.Base(matches[0])] = struct{}{}
	}

	return archives, expectedNames, nil
}

func requireRegularFile(path, description string) error {
	//nolint:gosec // path is the caller's validated dist path or a verified archive basename.
	info, err := os.Lstat(path)
	if err != nil {
		return failf("%s is not a regular non-symlink file: %s", description, path)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return failf("%s is not a regular non-symlink file: %s", description, path)
	}

	return nil
}

func readChecksums(manifest string, expected map[string]struct{}) ([]checksumEntry, error) {
	//nolint:gosec // manifest is the fixed checksums.txt under the caller's dist directory.
	file, err := os.Open(manifest)
	if err != nil {
		return nil, failf("missing regular checksums.txt in dist")
	}
	defer func() { _ = file.Close() }()

	linePattern := regexp.MustCompile(`^([0-9a-f]{64}) {2}(\*?)(.+)$`)
	entries := make([]checksumEntry, 0, len(expected))
	seen := make(map[string]struct{}, len(expected))
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, initialScanBuffer), maxChecksumLineBytes)

	for scanner.Scan() {
		line := scanner.Text()
		groups := linePattern.FindStringSubmatch(line)
		if groups == nil {
			return nil, failf("malformed SHA-256 checksum line: %s", line)
		}
		name := groups[3]
		if filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) {
			return nil, failf("checksum entry is not an archive basename: %s", name)
		}
		if _, ok := expected[name]; !ok {
			return nil, failf("checksum entry is not an expected Darwin archive: %s", name)
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, failf("duplicate checksum entry: %s", name)
		}
		seen[name] = struct{}{}
		entries = append(entries, checksumEntry{name: name, digest: groups[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: read checksums.txt: %w", errVerify, err)
	}
	if len(entries) != len(expected) {
		return nil, failf("checksums.txt must cover exactly the three Darwin archives")
	}
	for name := range expected {
		if _, ok := seen[name]; !ok {
			return nil, failf("checksums.txt is missing %s", name)
		}
	}

	return entries, nil
}

func verifyChecksums(directory string, entries []checksumEntry) error {
	for _, entry := range entries {
		actual, err := fileSHA256(filepath.Join(directory, entry.name))
		if err != nil {
			return err
		}
		if actual != entry.digest {
			return failf("SHA-256 verification failed for %s", entry.name)
		}
	}

	return nil
}

func fileSHA256(path string) (string, error) {
	//nolint:gosec // path is a verified archive or checksum basename under dist.
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%w: open %s: %w", errVerify, filepath.Base(path), err)
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("%w: hash %s: %w", errVerify, filepath.Base(path), err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func stageArchives(
	stage, manifest string,
	archives map[string]string,
) (map[string]string, error) {
	archiveDir := filepath.Join(stage, "archives")
	if err := os.Mkdir(archiveDir, privateDirMode); err != nil {
		return nil, fmt.Errorf("%w: create staged archive directory: %w", errVerify, err)
	}
	if err := copyRegularFile(manifest, filepath.Join(archiveDir, checksumName)); err != nil {
		return nil, err
	}

	staged := make(map[string]string, len(archives))
	for label, source := range archives {
		destination := filepath.Join(archiveDir, filepath.Base(source))
		if err := copyRegularFile(source, destination); err != nil {
			return nil, err
		}
		staged[label] = destination
	}

	return staged, nil
}

func copyRegularFile(source, destination string) error {
	if err := requireRegularFile(source, "source file"); err != nil {
		return err
	}
	//nolint:gosec // source is a verified path under the caller's dist directory.
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("%w: open %s: %w", errVerify, filepath.Base(source), err)
	}
	defer func() { _ = input.Close() }()

	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, privateFileMode)
	if err != nil {
		return fmt.Errorf("%w: create staged file %s: %w", errVerify, filepath.Base(destination), err)
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()

		return fmt.Errorf("%w: copy %s: %w", errVerify, filepath.Base(source), err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("%w: close staged file %s: %w", errVerify, filepath.Base(destination), err)
	}

	return nil
}

func verifyArchive(archive string, spec archiveSpec, stage string, out io.Writer, lipo lipoFunc) error {
	members, err := inspectArchive(archive)
	if err != nil {
		return fmt.Errorf("%w: inspect %s archive: %w", errVerify, spec.label, err)
	}
	if len(members) != 1 || members[0].name != expectedBinary {
		return failf("archive member list is not exactly %s: %s", expectedBinary, spec.label)
	}
	if !members[0].regular {
		return failf("archive member is not a regular file: %s", spec.label)
	}

	binaryPath := filepath.Join(stage, "extracted-"+spec.label, expectedBinary)
	if err := extractValidatedArchive(archive, filepath.Join(stage, "extracted-"+spec.label)); err != nil {
		return err
	}
	if err := requireRegularFile(binaryPath, "extracted "+expectedBinary); err != nil {
		return err
	}

	archOutput, err := lipo(binaryPath)
	if err != nil {
		return failf("lipo could not inspect %s: %v", spec.label, err)
	}
	trimmed := strings.TrimSpace(archOutput)
	if !architectureMatches(spec, trimmed) {
		return failf("%s has unexpected lipo architecture set: %s", spec.label, trimmed)
	}

	_, _ = fmt.Fprintf(out, "PASS %s: SHA-256, one regular %s, lipo -archs %s\n",
		spec.label, expectedBinary, trimmed)

	return nil
}

type memberInfo struct {
	name    string
	regular bool
}

func inspectArchive(path string) ([]memberInfo, error) {
	//nolint:gosec // path is a verified archive located under dist or private staging.
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: open archive: %w", errVerify, err)
	}
	defer func() { _ = file.Close() }()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("%w: open gzip stream: %w", errVerify, err)
	}
	defer func() { _ = gzipReader.Close() }()

	reader := tar.NewReader(gzipReader)
	members := make([]memberInfo, 0, 1)
	for {
		header, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("%w: read tar member: %w", errVerify, nextErr)
		}
		members = append(members, memberInfo{
			name:    header.Name,
			regular: header.Typeflag == tar.TypeReg,
		})
	}

	return members, nil
}

func extractValidatedArchive(archive, destination string) error {
	if err := os.Mkdir(destination, privateDirMode); err != nil {
		return fmt.Errorf("%w: create extraction directory: %w", errVerify, err)
	}

	//nolint:gosec // archive is a verified staged archive; destination is the private extraction root.
	file, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("%w: open archive for extraction: %w", errVerify, err)
	}
	defer func() { _ = file.Close() }()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("%w: open gzip stream: %w", errVerify, err)
	}
	defer func() { _ = gzipReader.Close() }()

	reader := tar.NewReader(gzipReader)
	header, err := reader.Next()
	if err != nil {
		return fmt.Errorf("%w: read validated member: %w", errVerify, err)
	}
	if header.Name != expectedBinary || header.Typeflag != tar.TypeReg {
		return failf("validated archive changed before extraction")
	}
	if header.Size < 0 || header.Size > maxArtifactBytes {
		return failf("archive member exceeds the extraction size bound")
	}

	// The destination basename is fixed; no archive-controlled path participates.

	output, err := os.OpenFile(
		filepath.Join(destination, expectedBinary),
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		privateFileMode,
	)
	if err != nil {
		return fmt.Errorf("%w: create extracted binary: %w", errVerify, err)
	}
	if _, err := io.CopyN(output, reader, header.Size); err != nil {
		_ = output.Close()

		return fmt.Errorf("%w: extract binary: %w", errVerify, err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("%w: close extracted binary: %w", errVerify, err)
	}

	return nil
}

func architectureMatches(spec archiveSpec, output string) bool {
	fields := strings.Fields(output)
	if !spec.universal {
		return len(fields) == 1 && fields[0] == spec.archs[0]
	}
	if len(fields) != len(spec.archs) {
		return false
	}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		seen[field] = struct{}{}
	}
	if len(seen) != len(spec.archs) {
		return false
	}
	for _, arch := range spec.archs {
		if _, ok := seen[arch]; !ok {
			return false
		}
	}

	return true
}

func archiveSpecs() []archiveSpec {
	return []archiveSpec{
		{label: "darwin_arm64", archs: []string{"arm64"}},
		{label: "darwin_amd64", archs: []string{"x86_64"}},
		{label: "darwin_universal", archs: []string{"arm64", "x86_64"}, universal: true},
	}
}

func lipoArchs(binary string) (string, error) {
	output, err := exec.CommandContext(context.Background(), "lipo", "-archs", binary).Output()
	if err != nil {
		return "", fmt.Errorf("run lipo: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func failf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errVerify, fmt.Sprintf(format, args...))
}
