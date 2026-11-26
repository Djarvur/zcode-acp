// Command scan-secrets rejects live-credential value shapes in tracked files.
// It preserves the former shell gate's regexes, tracked scope, and exit
// behavior while always redacting matched values from its output.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const maxFindings = 5

var errEmptyRoot = errors.New("git returned an empty worktree root")

type pattern struct {
	label string
	regex *regexp.Regexp
}

type finding struct {
	path   string
	line   int
	binary bool
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr))
}

func run(stdout, stderr io.Writer) int {
	root, err := repositoryRoot()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scan-secrets: %v\n", err)

		return 1
	}
	files, err := trackedFiles(root)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scan-secrets: %v\n", err)

		return 1
	}

	return scanAt(root, files, stdout)
}

func repositoryRoot() (string, error) {
	output, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git worktree: %w", err)
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", errEmptyRoot
	}

	return root, nil
}

func trackedFiles(root string) ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "ls-files", "-z")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	parts := bytes.Split(output, []byte{0})
	files := make([]string, 0, len(parts))
	for _, raw := range parts {
		path := string(raw)
		if path == "" || path == ".planning" || strings.HasPrefix(path, ".planning/") {
			continue
		}
		files = append(files, path)
	}

	return files, nil
}

func scanAt(root string, files []string, out io.Writer) int {
	failed := false
	for _, pattern := range credentialPatterns() {
		findings := findMatches(root, files, pattern.regex)
		if len(findings) == 0 {
			continue
		}
		failed = true
		printFindings(out, findings)
		_, _ = fmt.Fprintf(out, "SECRET SHAPE: %s\n", pattern.label)
	}
	if failed {
		return 1
	}

	return 0
}

func credentialPatterns() []pattern {
	return []pattern{
		{
			label: `eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`,
			regex: regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`),
		},
		{
			label: `enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}`,
			regex: regexp.MustCompile(`enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}`),
		},
		{
			label: `"(access_token|apiKey|api-key|token)"\s*:\s*"[A-Za-z0-9_-]{20,}`,
			regex: regexp.MustCompile(`"(access_token|apiKey|api-key|token)"\s*:\s*"[A-Za-z0-9_-]{20,}`),
		},
	}
}

func findMatches(root string, files []string, pattern *regexp.Regexp) []finding {
	findings := make([]finding, 0, maxFindings)
	allowlist := allowedLines()
	for _, relativePath := range files {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
		if err != nil {
			continue
		}
		if bytes.IndexByte(data, 0) >= 0 {
			if pattern.Match(data) {
				findings = append(findings, finding{path: relativePath, binary: true})
			}
			if len(findings) >= maxFindings {
				return findings
			}

			continue
		}

		for lineNumber, line := range strings.Split(string(data), "\n") {
			if !pattern.MatchString(line) || isAllowed(line, allowlist) {
				continue
			}
			findings = append(findings, finding{path: relativePath, line: lineNumber + 1})
			if len(findings) >= maxFindings {
				return findings
			}
		}
	}

	return findings
}

func allowedLines() []string {
	return []string{
		"enc:v2:os:fix1" + "-synthetic-not-a-real-credential",
		"zc-synthetic-fixture-not-a-real-key",
		"zc-synthetic-pat-a-corpus-fixture",
		"zc-synthetic-pat-b-corpus-fixture",
		"enc:v" + "2:os:3eb1bd439947eb762998e566:" +
			"MDEyMzQ1Njc4OWFi.Rwp6ooipuq_S1a9bRUi96w." +
			"_8CfTja4_TaGfE3J22EITiGrt71yrA8FUOaudeSW9t0HDQIYng",
	}
}

func isAllowed(line string, allowlist []string) bool {
	for _, marker := range allowlist {
		if strings.Contains(line, marker) {
			return true
		}
	}

	return false
}

func printFindings(out io.Writer, findings []finding) {
	for _, hit := range findings {
		if hit.binary {
			_, _ = fmt.Fprintf(out, "Binary file %s matches\n", hit.path)

			continue
		}
		_, _ = fmt.Fprintf(out, "%s:%d:[REDACTED]\n", hit.path, hit.line)
	}
}
