package acpfront

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSkillItems locks the catalog scanner (CAT-02): every direct child
// dir's SKILL.md frontmatter name/description becomes one item, entries
// sorted by name, unreadable/frontmatter-less children skipped silently,
// a missing root contributes nothing and never an error.
func TestSkillItems(t *testing.T) {
	roots := []string{filepath.Join("testdata", "catalog")}

	items := SkillItems(roots)

	if len(items) != 2 {
		t.Fatalf("skill items = %d (%v), want exactly 2 (the broken and SKILL-less children skipped)", len(items), items)
	}

	if items[0].Name != "gsd-manager" || items[0].Description != "Manage GSD milestones and phases from the chat" {
		t.Fatalf("item[0] = %+v, want the gsd-manager frontmatter (sorted first)", items[0])
	}

	if items[1].Name != "zed-tools" || items[1].Description != "Tool helper skills" {
		t.Fatalf("item[1] = %+v, want the zed-tools frontmatter", items[1])
	}

	if items := SkillItems([]string{filepath.Join("testdata", "catalog-missing")}); len(items) != 0 {
		t.Fatalf("a missing root produced %v, want no items and no error", items)
	}
}

// TestSkillItemsDirNameFallback locks the name fallback: a SKILL.md with
// frontmatter but no name line takes the child directory's name.
func TestSkillItemsDirNameFallback(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "unnamed-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\ndescription: only a description\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	items := SkillItems([]string{root})
	if len(items) != 1 || items[0].Name != "unnamed-skill" || items[0].Description != "only a description" {
		t.Fatalf("items = %+v, want the directory-name fallback with the description", items)
	}
}

// TestDefaultCatalogRoots locks the root resolution (CAT-02): the user
// skills dir and the installed plugin-caches glob, in that order.
func TestDefaultCatalogRoots(t *testing.T) {
	roots := DefaultCatalogRoots("/home/tester")

	if len(roots) != 2 {
		t.Fatalf("roots = %v, want exactly the user skills dir and the plugin-cache glob", roots)
	}

	wantUser := filepath.Join("/home/tester", ".zcode", "skills")
	wantPlugins := filepath.Join("/home/tester", ".zcode", "cli", "plugins", "cache", "*", "*", "*", "skills")

	if roots[0] != wantUser || roots[1] != wantPlugins {
		t.Fatalf("roots = %q, want %q then %q", roots, wantUser, wantPlugins)
	}
}
