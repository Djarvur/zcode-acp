package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const readmeFixture = "docs/protocol/README.md"

func writeFixture(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", relativePath, err)
	}
}

func runFixture(t *testing.T, root string, args ...string) (code int, output string) {
	t.Helper()
	var out bytes.Buffer
	code = runAt(root, args, &out)

	return code, out.String()
}

func TestConformanceReportsMissingDocsWithoutFailing(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, readmeFixture, "## Provenance legend\n")

	code, output := runFixture(t, root)
	if code != 0 {
		t.Fatalf("conformance exit = %d, want 0; output:\n%s", code, output)
	}
	if count := strings.Count(output, "PENDING: "); count != 5 {
		t.Fatalf("PENDING count = %d, want 5; output:\n%s", count, output)
	}
	if strings.Contains(output, "CHECK FAIL: ") {
		t.Fatalf("conformance unexpectedly failed:\n%s", output)
	}
}

func TestPhaseGateRequiresEveryDeliverableDoc(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, readmeFixture, "## Provenance legend\n")

	code, output := runFixture(t, root, phaseGateArg)
	if code != 1 {
		t.Fatalf("phase gate exit = %d, want 1; output:\n%s", code, output)
	}
	if !strings.Contains(output, "PENDING: docs/protocol/workspace-key.md") {
		t.Fatalf("missing deliverable was not reported:\n%s", output)
	}
}

func TestLegendMustExistOnceInREADME(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantError string
	}{
		{
			name:      "missing",
			files:     map[string]string{readmeFixture: "# README\n"},
			wantError: "provenance legend missing",
		},
		{
			name: "duplicate",
			files: map[string]string{
				readmeFixture:                    "## Provenance legend\n## Provenance legend\n",
				"docs/protocol/workspace-key.md": "## Provenance legend\n",
			},
			wantError: "provenance legend defined 3 times",
		},
		{
			name: "wrong file",
			files: map[string]string{
				readmeFixture:                    "# README\n",
				"docs/protocol/workspace-key.md": "## Provenance legend\n",
			},
			wantError: "must live only in " + readmeFixture,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for relativePath, content := range test.files {
				writeFixture(t, root, relativePath, content)
			}
			code, output := runFixture(t, root)
			if code != 1 || !strings.Contains(output, test.wantError) {
				t.Fatalf("exit = %d, want 1 with %q; output:\n%s", code, test.wantError, output)
			}
		})
	}
}

func TestExperimentTagsResolveAndCaptureHeadersStayCurrent(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, readmeFixture, "## Provenance legend\n")
	writeFixture(t, root, "docs/research/evidence.md", "[V-EXP EXP-SAMPLE-01]\n")

	code, output := runFixture(t, root)
	if code != 1 || !strings.Contains(output, "V-EXP id does not resolve") {
		t.Fatalf("missing capture accepted: exit %d; output:\n%s", code, output)
	}

	writeFixture(t, root, "experiments/captures/EXP-SAMPLE-01.jsonl", `{"capture":"EXP-SAMPLE-01"}`+"\n")
	code, output = runFixture(t, root)
	if code != 0 {
		t.Fatalf("valid capture rejected: exit %d; output:\n%s", code, output)
	}

	writeFixture(t, root, "experiments/captures/EXP-SAMPLE-01.jsonl", `{"capture":"OTHER"}`+"\n")
	code, output = runFixture(t, root)
	if code != 1 || !strings.Contains(output, "capture header missing or stale") {
		t.Fatalf("stale capture header accepted: exit %d; output:\n%s", code, output)
	}
}

func TestMalformedProvenanceTagsAreRejected(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, readmeFixture, "## Provenance legend\n[V-BOGUS value]\n")

	code, output := runFixture(t, root)
	if code != 1 || !strings.Contains(output, "malformed provenance tag") {
		t.Fatalf("malformed tag accepted: exit %d; output:\n%s", code, output)
	}
}

func TestPhaseGateChecksRequiredHeadings(t *testing.T) {
	root := t.TempDir()
	for relativePath, headings := range requiredSections() {
		writeFixture(t, root, relativePath, strings.Join(headings, "\n")+"\n")
	}

	code, output := runFixture(t, root, phaseGateArg)
	if code != 0 {
		t.Fatalf("complete fixture rejected: exit %d; output:\n%s", code, output)
	}

	writeFixture(t, root, "docs/protocol/workspace-key.md", "## Wire schema\n")
	code, output = runFixture(t, root, phaseGateArg)
	if code != 1 || !strings.Contains(output, "missing required section heading: ## Derivation convention") {
		t.Fatalf("missing heading accepted: exit %d; output:\n%s", code, output)
	}
}

func TestUserHomePathAndKeyMintingScans(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, readmeFixture, "## Provenance legend\n")
	writeFixture(t, root, "experiments/demo.mjs", `const endpoint = "api_`+`keys";`+"\n")

	code, output := runFixture(t, root)
	if code != 1 || !strings.Contains(output, "key-minting endpoint reference") {
		t.Fatalf("key-minting reference accepted: exit %d; output:\n%s", code, output)
	}

	writeFixture(t, root, "experiments/demo.mjs", "const endpoint = \"safe\";\n")
	writeFixture(t, root, "docs/extra.md", "/home/example/project\n")
	code, output = runFixture(t, root)
	if code != 1 || !strings.Contains(output, "absolute path under a user home directory") {
		t.Fatalf("user-home path accepted: exit %d; output:\n%s", code, output)
	}
}

func TestFindRepoRootWalksToWorktreeRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatalf("create .git marker: %v", err)
	}
	nested := filepath.Join(root, "docs", "protocol")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}

	got, err := findRepoRoot(nested)
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	if got != root {
		t.Fatalf("findRepoRoot = %q, want %q", got, root)
	}
}
