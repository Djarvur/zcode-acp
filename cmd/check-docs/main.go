// Command check-docs enforces documentation structure and provenance-tag
// conformance. It replaces the former Bash gate with portable Go filesystem
// APIs and preserves the CHECK FAIL / PENDING stdout and exit contract.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	phaseGateArg   = "--phase-gate"
	maxPrinted     = 5
	readmeDoc      = "docs/protocol/README.md"
	sourcesHeading = "## Sources & provenance"
)

var errNotWorktree = errors.New("not inside a git worktree")

type gateMode int

const (
	conformanceMode gateMode = iota
	phaseGateMode
)

type checker struct {
	root       string
	out        io.Writer
	mode       gateMode
	failed     bool
	docsPaths  []string
	docsLoaded bool
	docsErr    error
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cwd, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check-docs: cannot determine working directory: %v\n", err)

		return 1
	}

	root, err := findRepoRoot(cwd)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "check-docs: %v\n", err)

		return 1
	}

	return runAt(root, args, stdout)
}

func runAt(root string, args []string, stdout io.Writer) int {
	check := &checker{root: root, out: stdout}
	if len(args) > 0 && args[0] == phaseGateArg {
		check.mode = phaseGateMode
	}

	check.checkMissingDocs()
	if err := check.checkLegend(); err != nil {
		check.failf("cannot scan documentation: %v", err)
	}
	if err := check.checkExperimentTags(); err != nil {
		check.failf("cannot scan provenance tags: %v", err)
	}
	if err := check.checkTagGrammar(); err != nil {
		check.failf("cannot scan provenance-tag grammar: %v", err)
	}
	if check.mode == phaseGateMode {
		if err := check.checkRequiredSections(); err != nil {
			check.failf("cannot scan required sections: %v", err)
		}
	}
	if err := check.checkUserHomePaths(); err != nil {
		check.failf("cannot scan committed paths: %v", err)
	}
	if err := check.checkKeyMinting(); err != nil {
		check.failf("cannot scan experiment scripts: %v", err)
	}

	if check.failed {
		return 1
	}

	return 0
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errNotWorktree
		}
		dir = parent
	}
}

func (c *checker) failf(format string, args ...any) {
	c.failed = true
	_, _ = fmt.Fprintf(c.out, "CHECK FAIL: "+format+"\n", args...)
}

func (c *checker) checkMissingDocs() {
	docs := requiredDocs()
	present := 0
	for _, rel := range docs {
		if info, err := os.Stat(filepath.Join(c.root, rel)); err == nil && info.Mode().IsRegular() {
			present++
		}
	}

	for _, rel := range docs {
		info, err := os.Stat(filepath.Join(c.root, rel))
		if err == nil && info.Mode().IsRegular() {
			continue
		}
		_, _ = fmt.Fprintf(c.out, "PENDING: %s (%d/6 present)\n", rel, present)
		if c.mode == phaseGateMode {
			c.failed = true
		}
	}
}

func (c *checker) checkLegend() error {
	paths, err := c.markdownDocs()
	if err != nil {
		return err
	}

	total := 0
	lastFile := ""
	for _, rel := range paths {
		lines, readErr := c.readLines(rel)
		if readErr != nil {
			return readErr
		}
		count := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "## Provenance legend") {
				count++
			}
		}
		if count > 0 {
			total += count
			lastFile = rel
		}
	}

	switch {
	case total == 0:
		c.failf("provenance legend missing: expected exactly one '## Provenance legend' heading, " +
			"in " + readmeDoc)
	case total > 1:
		c.failf("provenance legend defined %d times; it must be defined exactly once (%s), "+
			"other docs reference the README", total, readmeDoc)
	case lastFile != readmeDoc:
		c.failf("provenance legend must live only in %s (found in: %s)", readmeDoc, lastFile)
	}

	return nil
}

func (c *checker) checkExperimentTags() error {
	paths, err := c.markdownDocs()
	if err != nil {
		return err
	}
	tagPattern := regexp.MustCompile(`\[V-EXP [^]]+\]`)
	tags := map[string]struct{}{}

	for _, rel := range paths {
		lines, readErr := c.readLines(rel)
		if readErr != nil {
			return readErr
		}
		for _, line := range lines {
			for _, tag := range tagPattern.FindAllString(line, -1) {
				tags[tag] = struct{}{}
			}
		}
	}

	sorted := sortedKeys(tags)
	for _, tag := range sorted {
		id := strings.TrimSuffix(strings.TrimPrefix(tag, "[V-EXP "), "]")
		if strings.HasPrefix(id, "<") {
			continue
		}
		if err := c.checkExperimentTag(tag, id); err != nil {
			return err
		}
	}

	return nil
}

func (c *checker) checkExperimentTag(tag, id string) error {
	matches, _ := filepath.Glob(filepath.Join(c.root, "experiments", "captures", id+"*"))
	if len(matches) == 0 {
		c.failf("V-EXP id does not resolve: %s -> expected experiments/captures/%s*", tag, id)

		return nil
	}

	for _, capture := range matches {
		info, statErr := os.Stat(capture)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		firstLine, readErr := firstLine(capture)
		if readErr != nil || !strings.Contains(firstLine, `"capture":"`+id+`"`) {
			c.failf(`capture header missing or stale: %s first line must contain "capture":"%s" (T-1-07)`,
				c.relative(capture), id)
		}
	}

	return nil
}

func (c *checker) checkTagGrammar() error {
	paths, err := c.markdownDocs()
	if err != nil {
		return err
	}
	tagPattern := regexp.MustCompile(`\[V-[A-Za-z]+[^]]*\]`)
	validPattern := regexp.MustCompile(`^\[V-(SRC|EXP|RPT) [^]]+\]$`)
	bad := map[string]struct{}{}

	for _, rel := range paths {
		lines, readErr := c.readLines(rel)
		if readErr != nil {
			return readErr
		}
		for _, line := range lines {
			for _, tag := range tagPattern.FindAllString(line, -1) {
				if !validPattern.MatchString(tag) {
					bad[tag] = struct{}{}
				}
			}
		}
	}

	for _, tag := range sortedKeys(bad) {
		c.failf("malformed provenance tag (need [V-SRC|V-EXP|V-RPT <non-empty arg>]): %s", tag)
	}

	return nil
}

func (c *checker) checkRequiredSections() error {
	sections := requiredSections()
	for _, rel := range requiredDocs() {
		info, err := os.Stat(filepath.Join(c.root, rel))
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		lines, readErr := c.readLines(rel)
		if readErr != nil {
			return readErr
		}
		exact := make(map[string]struct{}, len(lines))
		for _, line := range lines {
			exact[line] = struct{}{}
		}
		for _, heading := range sections[rel] {
			if _, ok := exact[heading]; !ok {
				c.failf("%s missing required section heading: %s", rel, heading)
			}
		}
	}

	return nil
}

func (c *checker) checkUserHomePaths() error {
	pathPattern := regexp.MustCompile(`/(home|Users|root)/[A-Za-z0-9._-]+`)
	hits, err := c.scanFiles([]string{"docs", "experiments"}, nil, []*regexp.Regexp{pathPattern})
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		return nil
	}
	printHits(c.out, hits)
	c.failf("absolute path under a user home directory committed " +
		"(neutral-phrasing rule; use tilde or <placeholder> forms)")

	return nil
}

func (c *checker) checkKeyMinting() error {
	mintPattern := regexp.MustCompile(`(?i)api_keys`)
	include := func(_ string, entry fs.DirEntry) bool {
		ext := filepath.Ext(entry.Name())

		return ext == ".mjs" || ext == ".sh"
	}
	hits, err := c.scanFiles([]string{"experiments"}, include, []*regexp.Regexp{mintPattern})
	if err != nil {
		return err
	}
	if len(hits) == 0 {
		return nil
	}
	printHits(c.out, hits)
	c.failf("key-minting endpoint reference (api_keys) in experiments/ scripts (T-1-03: no flow may mint credentials)")

	return nil
}

func (c *checker) scanFiles(
	directories []string,
	include func(string, fs.DirEntry) bool,
	patterns []*regexp.Regexp,
) ([]string, error) {
	hits := make([]string, 0)
	for _, directory := range directories {
		root := filepath.Join(c.root, directory)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			return c.scanWalkEntry(path, entry, walkErr, include, patterns, &hits)
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("walk %s: %w", directory, err)
		}
	}

	return hits, nil
}

func (c *checker) scanWalkEntry(
	path string,
	entry fs.DirEntry,
	walkErr error,
	include func(string, fs.DirEntry) bool,
	patterns []*regexp.Regexp,
	hits *[]string,
) error {
	if walkErr != nil {
		if errors.Is(walkErr, fs.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("visit %s: %w", path, walkErr)
	}
	if entry.IsDir() || !entry.Type().IsRegular() {
		return nil
	}
	if include != nil && !include(path, entry) {
		return nil
	}

	lines, err := c.readAbsoluteLines(path)
	if err != nil {
		return err
	}
	rel := c.relative(path)
	for lineNumber, line := range lines {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				*hits = append(*hits, fmt.Sprintf("%s:%d:%s", rel, lineNumber+1, line))

				break
			}
		}
	}

	return nil
}

func (c *checker) markdownDocs() ([]string, error) {
	if c.docsLoaded {
		return c.docsPaths, c.docsErr
	}
	c.docsLoaded = true

	root := filepath.Join(c.root, "docs")
	paths := make([]string, 0)
	c.docsErr = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}

			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".md" {
			return nil
		}
		paths = append(paths, c.relative(path))

		return nil
	})
	if c.docsErr != nil && !errors.Is(c.docsErr, fs.ErrNotExist) {
		return nil, c.docsErr
	}
	sort.Strings(paths)
	c.docsPaths = paths

	return c.docsPaths, nil
}

func (c *checker) readLines(relativePath string) ([]string, error) {
	return c.readAbsoluteLines(filepath.Join(c.root, relativePath))
}

func (c *checker) readAbsoluteLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", c.relative(path), err)
	}
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")

	return strings.Split(normalized, "\n"), nil
}

func (c *checker) relative(path string) string {
	rel, err := filepath.Rel(c.root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	return filepath.ToSlash(rel)
}

func firstLine(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open capture: %w", err)
	}
	defer func() { _ = file.Close() }()

	line, err := bufio.NewReader(file).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read capture header: %w", err)
	}

	return strings.TrimSuffix(line, "\n"), nil
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

func printHits(out io.Writer, hits []string) {
	limit := min(len(hits), maxPrinted)
	for _, hit := range hits[:limit] {
		_, _ = fmt.Fprintln(out, hit)
	}
}

func requiredDocs() []string {
	return []string{
		readmeDoc,
		"docs/protocol/app-server-dialect.md",
		"docs/protocol/acp-client-dialect.md",
		"docs/protocol/install-discovery.md",
		"docs/protocol/workspace-key.md",
		"docs/research/reference-adapters.md",
	}
}

func requiredSections() map[string][]string {
	return map[string][]string{
		readmeDoc: {
			"## Provenance legend",
		},
		"docs/protocol/workspace-key.md": {
			"## Wire schema",
			"## Derivation convention",
			"## Experiment record",
			"## Host-freedom note",
			sourcesHeading,
		},
		"docs/protocol/app-server-dialect.md": {
			"## 1. Transport facts",
			"## 2. Method map",
			"## 3. Server-to-host surface",
			"## 4. session/subscribe and event-stream availability",
			"## 5. Turn-completion terminal signals",
			"## 6. Error-code table",
			"## 7. Session lifecycle methods",
			"## 8. Snapshot semantics",
			"## 9. 3.17 vs 3.18.2 drift baseline",
			sourcesHeading,
		},
		"docs/protocol/acp-client-dialect.md": {
			"## Preset-agent identity",
			"## Card registry of this build",
			"## Launch contract",
			"## Version floor",
			"## Load-bearing method set",
			"## Full ACP method range the client sends",
			"## String contracts",
			"## Client capabilities",
			"## Open items",
			sourcesHeading,
		},
		"docs/protocol/install-discovery.md": {
			"## Layout matrix",
			"## Locating the entrypoint per layout",
			"## Desktop entry and URL handler",
			"## Auto-update and the builtin revision",
			"## Recommended discovery order",
			sourcesHeading,
		},
		"docs/research/reference-adapters.md": {
			"## What the reference covers",
			"## Adopt",
			"## Avoid",
			"## Attribution",
			"## Weaker adapters",
			sourcesHeading,
		},
	}
}
